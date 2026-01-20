package airdrop

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sync"

	"github.com/google/uuid"
)

type SecureServer struct {
	port        int
	downloadDir string
	identity    *DeviceIdentity
	deviceName  string
	onRequest   func(req HandshakeRequest) bool
	onProgress  func(filename string, received, total int64)
	server      *http.Server
	sessions    map[string]*TransferSession
	mu          sync.Mutex
}

type TransferSession struct {
	SessionID      string
	SenderName     string
	Fingerprint    string
	Metadata       FileMetadata
	SessionKey     []byte
	TotalChunks    int
	ReceivedChunks map[int]bool
	FilePath       string
	File           *os.File
}

func NewSecureServer(port int, downloadDir, deviceName string) (*SecureServer, error) {
	// Load or generate device identity
	homeDir, _ := os.UserHomeDir()
	identityDir := filepath.Join(homeDir, ".sfm", "airdrop")

	identity, err := LoadOrGenerateIdentity(identityDir)
	if err != nil {
		return nil, fmt.Errorf("failed to load identity: %w", err)
	}

	log.Printf("Device fingerprint: %s", identity.Fingerprint)

	return &SecureServer{
		port:        port,
		downloadDir: downloadDir,
		identity:    identity,
		deviceName:  deviceName,
		sessions:    make(map[string]*TransferSession),
		onRequest: func(req HandshakeRequest) bool {
			return true // Auto-accept by default
		},
	}, nil
}

func (s *SecureServer) SetRequestHandler(handler func(req HandshakeRequest) bool) {
	s.onRequest = handler
}

func (s *SecureServer) SetProgressHandler(handler func(filename string, received, total int64)) {
	s.onProgress = handler
}

func (s *SecureServer) Start() error {
	if err := os.MkdirAll(s.downloadDir, 0755); err != nil {
		return fmt.Errorf("failed to create download directory: %w", err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/handshake", s.handleHandshake)
	mux.HandleFunc("/chunk", s.handleChunk)
	mux.HandleFunc("/status", s.handleStatus)
	mux.HandleFunc("/ping", s.handlePing)

	s.server = &http.Server{
		Addr:    fmt.Sprintf(":%d", s.port),
		Handler: mux,
	}

	log.Printf("Secure AirDrop server listening on port %d", s.port)
	return s.server.ListenAndServe()
}

func (s *SecureServer) Stop() error {
	if s.server != nil {
		return s.server.Close()
	}
	return nil
}

func (s *SecureServer) handlePing(w http.ResponseWriter, r *http.Request) {
	response := map[string]string{
		"device_name": s.deviceName,
		"fingerprint": s.identity.Fingerprint,
	}
	json.NewEncoder(w).Encode(response)
}

func (s *SecureServer) handleHandshake(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req HandshakeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// Verify signature (simplified - in production would verify against stored public keys)
	log.Printf("Handshake from: %s (%s)", req.DeviceName, req.DeviceFingerprint)
	log.Printf("File: %s (%d bytes)", req.FileMetadata.Name, req.FileMetadata.Size)

	// Ask user to accept/reject
	accepted := s.onRequest(req)

	if !accepted {
		resp := HandshakeResponse{
			Accepted: false,
			Message:  "Transfer rejected by user",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
		return
	}

	// Generate ephemeral key for ECDH
	privKey, pubKey, err := GenerateEphemeralKey()
	if err != nil {
		http.Error(w, "Failed to generate key", http.StatusInternalServerError)
		return
	}

	// Derive shared secret
	sessionKey, err := DeriveSharedSecret(privKey, req.EphemeralPubKey)
	if err != nil {
		http.Error(w, "Failed to derive session key", http.StatusInternalServerError)
		return
	}

	// Create session
	sessionID := uuid.New().String()
	totalChunks := int(req.FileMetadata.Size / (4 * 1024 * 1024))
	if req.FileMetadata.Size%(4*1024*1024) != 0 {
		totalChunks++
	}

	session := &TransferSession{
		SessionID:      sessionID,
		SenderName:     req.DeviceName,
		Fingerprint:    req.DeviceFingerprint,
		Metadata:       req.FileMetadata,
		SessionKey:     sessionKey,
		TotalChunks:    totalChunks,
		ReceivedChunks: make(map[int]bool),
		FilePath:       filepath.Join(s.downloadDir, req.FileMetadata.Name),
	}

	// Create output file
	file, err := os.Create(session.FilePath)
	if err != nil {
		http.Error(w, "Failed to create file", http.StatusInternalServerError)
		return
	}
	session.File = file

	s.mu.Lock()
	s.sessions[sessionID] = session
	s.mu.Unlock()

	// Send response
	resp := HandshakeResponse{
		Accepted:        true,
		EphemeralPubKey: pubKey,
		SessionID:       sessionID,
		Message:         "Transfer accepted",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)

	log.Printf("Session created: %s", sessionID)
}

func (s *SecureServer) handleChunk(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Read chunk metadata from header
	var metadata ChunkMetadata
	metadataStr := r.Header.Get("X-Chunk-Metadata")
	if err := json.Unmarshal([]byte(metadataStr), &metadata); err != nil {
		http.Error(w, "Invalid chunk metadata", http.StatusBadRequest)
		return
	}

	// Get session
	s.mu.Lock()
	session, exists := s.sessions[metadata.SessionID]
	s.mu.Unlock()

	if !exists {
		http.Error(w, "Invalid session", http.StatusBadRequest)
		return
	}

	// Read encrypted chunk
	encryptedData, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read chunk", http.StatusInternalServerError)
		return
	}

	// Decrypt chunk
	decryptedData, err := DecryptChunk(encryptedData, session.SessionKey)
	if err != nil {
		http.Error(w, "Failed to decrypt chunk", http.StatusInternalServerError)
		return
	}

	// Verify checksum
	checksum := CalculateChunkChecksum(decryptedData)
	if checksum != metadata.Checksum {
		ack := ChunkAck{
			Index:     metadata.Index,
			SessionID: metadata.SessionID,
			Success:   false,
			Error:     "Checksum mismatch",
		}
		json.NewEncoder(w).Encode(ack)
		return
	}

	// Write chunk to file
	offset := int64(metadata.Index) * (4 * 1024 * 1024)
	if _, err := session.File.WriteAt(decryptedData, offset); err != nil {
		ack := ChunkAck{
			Index:     metadata.Index,
			SessionID: metadata.SessionID,
			Success:   false,
			Error:     "Failed to write chunk",
		}
		json.NewEncoder(w).Encode(ack)
		return
	}

	// Mark chunk as received
	s.mu.Lock()
	session.ReceivedChunks[metadata.Index] = true
	received := len(session.ReceivedChunks)
	s.mu.Unlock()

	// Update progress
	if s.onProgress != nil {
		progress := float64(received) / float64(session.TotalChunks) * 100
		s.onProgress(session.Metadata.Name, int64(received), int64(session.TotalChunks))
		log.Printf("Progress: %.2f%% (%d/%d chunks)", progress, received, session.TotalChunks)
	}

	// Send ACK
	ack := ChunkAck{
		Index:     metadata.Index,
		SessionID: metadata.SessionID,
		Success:   true,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ack)

	// Check if transfer complete
	if received == session.TotalChunks {
		session.File.Close()
		log.Printf("✓ Transfer complete: %s", session.FilePath)

		s.mu.Lock()
		delete(s.sessions, metadata.SessionID)
		s.mu.Unlock()
	}
}

func (s *SecureServer) handleStatus(w http.ResponseWriter, r *http.Request) {
	sessionID := r.URL.Query().Get("session_id")

	s.mu.Lock()
	session, exists := s.sessions[sessionID]
	s.mu.Unlock()

	if !exists {
		http.Error(w, "Session not found", http.StatusNotFound)
		return
	}

	receivedChunks := make([]int, 0, len(session.ReceivedChunks))
	for idx := range session.ReceivedChunks {
		receivedChunks = append(receivedChunks, idx)
	}

	status := TransferStatus{
		SessionID:      sessionID,
		TotalChunks:    session.TotalChunks,
		ReceivedChunks: receivedChunks,
		Progress:       float64(len(receivedChunks)) / float64(session.TotalChunks) * 100,
		CanResume:      true,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

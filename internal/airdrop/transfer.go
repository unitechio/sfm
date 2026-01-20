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
)

type FileMetadata struct {
	Name string `json:"name"`
	Size int64  `json:"size"`
	Mime string `json:"mime"`
}

type TransferRequest struct {
	Metadata FileMetadata `json:"metadata"`
	SenderIP string       `json:"sender_ip"`
}

type TransferResponse struct {
	Accepted bool   `json:"accepted"`
	Message  string `json:"message,omitempty"`
}

type Server struct {
	port           int
	downloadDir    string
	onRequest      func(req TransferRequest) bool // Callback for accept/reject
	onProgress     func(filename string, received, total int64)
	server         *http.Server
	mu             sync.Mutex
	activeTransfer bool
}

func NewServer(port int, downloadDir string) *Server {
	return &Server{
		port:        port,
		downloadDir: downloadDir,
		onRequest: func(req TransferRequest) bool {
			// Auto-accept by default
			return true
		},
	}
}

// SetRequestHandler sets the callback for handling transfer requests
func (s *Server) SetRequestHandler(handler func(req TransferRequest) bool) {
	s.onRequest = handler
}

// SetProgressHandler sets the callback for progress updates
func (s *Server) SetProgressHandler(handler func(filename string, received, total int64)) {
	s.onProgress = handler
}

// Start starts the HTTP server
func (s *Server) Start() error {
	if err := os.MkdirAll(s.downloadDir, 0755); err != nil {
		return fmt.Errorf("failed to create download directory: %w", err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/request", s.handleRequest)
	mux.HandleFunc("/send", s.handleSend)
	mux.HandleFunc("/ping", s.handlePing)

	s.server = &http.Server{
		Addr:    fmt.Sprintf(":%d", s.port),
		Handler: mux,
	}

	log.Printf("AirDrop server listening on port %d", s.port)
	return s.server.ListenAndServe()
}

// Stop stops the HTTP server
func (s *Server) Stop() error {
	if s.server != nil {
		return s.server.Close()
	}
	return nil
}

func (s *Server) handlePing(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("pong"))
}

func (s *Server) handleRequest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req TransferRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// Get sender IP
	req.SenderIP = r.RemoteAddr

	// Check if already transferring
	s.mu.Lock()
	if s.activeTransfer {
		s.mu.Unlock()
		resp := TransferResponse{
			Accepted: false,
			Message:  "Busy with another transfer",
		}
		json.NewEncoder(w).Encode(resp)
		return
	}
	s.mu.Unlock()

	// Ask user to accept/reject
	accepted := s.onRequest(req)

	resp := TransferResponse{
		Accepted: accepted,
	}

	if accepted {
		s.mu.Lock()
		s.activeTransfer = true
		s.mu.Unlock()
		resp.Message = "Transfer accepted"
	} else {
		resp.Message = "Transfer rejected"
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (s *Server) handleSend(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	defer func() {
		s.mu.Lock()
		s.activeTransfer = false
		s.mu.Unlock()
	}()

	// Get metadata from headers
	filename := r.Header.Get("X-File-Name")
	if filename == "" {
		http.Error(w, "Missing filename", http.StatusBadRequest)
		return
	}

	// Create output file
	outputPath := filepath.Join(s.downloadDir, filename)
	outFile, err := os.Create(outputPath)
	if err != nil {
		http.Error(w, "Failed to create file", http.StatusInternalServerError)
		return
	}
	defer outFile.Close()

	// Stream file with progress
	received := int64(0)
	total := r.ContentLength
	buffer := make([]byte, 32*1024) // 32KB buffer

	for {
		n, err := r.Body.Read(buffer)
		if n > 0 {
			if _, writeErr := outFile.Write(buffer[:n]); writeErr != nil {
				http.Error(w, "Failed to write file", http.StatusInternalServerError)
				return
			}
			received += int64(n)

			if s.onProgress != nil {
				s.onProgress(filename, received, total)
			}
		}

		if err == io.EOF {
			break
		}
		if err != nil {
			http.Error(w, "Failed to read file", http.StatusInternalServerError)
			return
		}
	}

	log.Printf("Received file: %s (%d bytes)", filename, received)
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

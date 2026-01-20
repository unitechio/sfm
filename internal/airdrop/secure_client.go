package airdrop

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

type SecureClient struct {
	httpClient *http.Client
	identity   *DeviceIdentity
	deviceName string
}

func NewSecureClient(deviceName string) (*SecureClient, error) {
	// Load or generate device identity
	homeDir, _ := os.UserHomeDir()
	identityDir := filepath.Join(homeDir, ".sfm", "airdrop")

	identity, err := LoadOrGenerateIdentity(identityDir)
	if err != nil {
		return nil, fmt.Errorf("failed to load identity: %w", err)
	}

	log.Printf("Client fingerprint: %s", identity.Fingerprint)

	return &SecureClient{
		httpClient: &http.Client{
			Timeout: 10 * time.Minute,
		},
		identity:   identity,
		deviceName: deviceName,
	}, nil
}

func (c *SecureClient) SendFile(targetIP string, targetPort int, filePath string, onProgress func(sent, total int64)) error {
	// Open file
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	fileInfo, err := file.Stat()
	if err != nil {
		return fmt.Errorf("failed to stat file: %w", err)
	}

	// Generate ephemeral key for ECDH
	privKey, pubKey, err := GenerateEphemeralKey()
	if err != nil {
		return fmt.Errorf("failed to generate ephemeral key: %w", err)
	}

	// Create file metadata
	metadata := FileMetadata{
		Name: filepath.Base(filePath),
		Size: fileInfo.Size(),
		Mime: "application/octet-stream",
	}

	// Create handshake request
	handshakeReq, err := CreateHandshakeRequest(c.identity, c.deviceName, pubKey, metadata)
	if err != nil {
		return fmt.Errorf("failed to create handshake: %w", err)
	}

	// Send handshake
	handshakeURL := fmt.Sprintf("http://%s:%d/handshake", targetIP, targetPort)
	handshakeBody, _ := json.Marshal(handshakeReq)

	resp, err := c.httpClient.Post(handshakeURL, "application/json", bytes.NewReader(handshakeBody))
	if err != nil {
		return fmt.Errorf("failed to send handshake: %w", err)
	}
	defer resp.Body.Close()

	var handshakeResp HandshakeResponse
	if err := json.NewDecoder(resp.Body).Decode(&handshakeResp); err != nil {
		return fmt.Errorf("failed to decode handshake response: %w", err)
	}

	if !handshakeResp.Accepted {
		return fmt.Errorf("transfer rejected: %s", handshakeResp.Message)
	}

	log.Printf("Handshake accepted. Session ID: %s", handshakeResp.SessionID)

	// Derive shared secret
	sessionKey, err := DeriveSharedSecret(privKey, handshakeResp.EphemeralPubKey)
	if err != nil {
		return fmt.Errorf("failed to derive session key: %w", err)
	}

	// Calculate total chunks
	chunkSize := int64(4 * 1024 * 1024) // 4MB
	totalChunks := int(fileInfo.Size() / chunkSize)
	if fileInfo.Size()%chunkSize != 0 {
		totalChunks++
	}

	log.Printf("Sending %d chunks...", totalChunks)

	// Send chunks
	buffer := make([]byte, chunkSize)
	for chunkIndex := 0; chunkIndex < totalChunks; chunkIndex++ {
		// Read chunk
		n, err := file.Read(buffer)
		if err != nil && err != io.EOF {
			return fmt.Errorf("failed to read chunk %d: %w", chunkIndex, err)
		}

		chunkData := buffer[:n]

		// Calculate checksum
		checksum := CalculateChunkChecksum(chunkData)

		// Encrypt chunk
		encryptedChunk, err := EncryptChunk(chunkData, sessionKey)
		if err != nil {
			return fmt.Errorf("failed to encrypt chunk %d: %w", chunkIndex, err)
		}

		// Create chunk metadata
		chunkMetadata := ChunkMetadata{
			Index:     chunkIndex,
			Total:     totalChunks,
			Size:      n,
			Checksum:  checksum,
			SessionID: handshakeResp.SessionID,
		}

		// Send chunk
		if err := c.sendChunk(targetIP, targetPort, chunkMetadata, encryptedChunk); err != nil {
			return fmt.Errorf("failed to send chunk %d: %w", chunkIndex, err)
		}

		// Update progress
		if onProgress != nil {
			onProgress(int64(chunkIndex+1), int64(totalChunks))
		}
	}

	log.Printf("✓ All chunks sent successfully")
	return nil
}

func (c *SecureClient) sendChunk(targetIP string, targetPort int, metadata ChunkMetadata, encryptedData []byte) error {
	chunkURL := fmt.Sprintf("http://%s:%d/chunk", targetIP, targetPort)

	// Create request
	req, err := http.NewRequest(http.MethodPost, chunkURL, bytes.NewReader(encryptedData))
	if err != nil {
		return err
	}

	// Add metadata to header
	metadataJSON, _ := json.Marshal(metadata)
	req.Header.Set("X-Chunk-Metadata", string(metadataJSON))
	req.Header.Set("Content-Type", "application/octet-stream")

	// Send request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Read ACK
	var ack ChunkAck
	if err := json.NewDecoder(resp.Body).Decode(&ack); err != nil {
		return fmt.Errorf("failed to decode ACK: %w", err)
	}

	if !ack.Success {
		return fmt.Errorf("chunk rejected: %s", ack.Error)
	}

	return nil
}

func (c *SecureClient) GetTransferStatus(targetIP string, targetPort int, sessionID string) (*TransferStatus, error) {
	statusURL := fmt.Sprintf("http://%s:%d/status?session_id=%s", targetIP, targetPort, sessionID)

	resp, err := c.httpClient.Get(statusURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var status TransferStatus
	if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
		return nil, err
	}

	return &status, nil
}

package airdrop

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
)

// HandshakeRequest is sent by sender to initiate transfer
type HandshakeRequest struct {
	DeviceName        string       `json:"device_name"`
	DeviceFingerprint string       `json:"device_fingerprint"`
	EphemeralPubKey   []byte       `json:"ephemeral_pubkey"`
	FileMetadata      FileMetadata `json:"file_metadata"`
	Signature         []byte       `json:"signature"`
}

// HandshakeResponse is sent by receiver
type HandshakeResponse struct {
	Accepted        bool   `json:"accepted"`
	EphemeralPubKey []byte `json:"ephemeral_pubkey,omitempty"`
	SessionID       string `json:"session_id,omitempty"`
	Message         string `json:"message,omitempty"`
}

// ChunkMetadata represents a file chunk
type ChunkMetadata struct {
	Index     int    `json:"index"`
	Total     int    `json:"total"`
	Size      int    `json:"size"`
	Checksum  string `json:"checksum"`
	SessionID string `json:"session_id"`
}

// ChunkAck acknowledges chunk receipt
type ChunkAck struct {
	Index     int    `json:"index"`
	SessionID string `json:"session_id"`
	Success   bool   `json:"success"`
	Error     string `json:"error,omitempty"`
}

// TransferStatus represents transfer state
type TransferStatus struct {
	SessionID      string  `json:"session_id"`
	TotalChunks    int     `json:"total_chunks"`
	ReceivedChunks []int   `json:"received_chunks"`
	Progress       float64 `json:"progress"`
	CanResume      bool    `json:"can_resume"`
}

// CalculateChunkChecksum computes SHA256 of chunk data
func CalculateChunkChecksum(data []byte) string {
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

// CreateHandshakeRequest creates a signed handshake request
func CreateHandshakeRequest(identity *DeviceIdentity, deviceName string, ephemeralPubKey []byte, metadata FileMetadata) (*HandshakeRequest, error) {
	req := &HandshakeRequest{
		DeviceName:        deviceName,
		DeviceFingerprint: identity.Fingerprint,
		EphemeralPubKey:   ephemeralPubKey,
		FileMetadata:      metadata,
	}

	// Sign the request
	data, _ := json.Marshal(req)
	req.Signature = identity.Sign(data)

	return req, nil
}

// VerifyHandshakeRequest verifies the handshake signature
func VerifyHandshakeRequest(req *HandshakeRequest, pubKey []byte) bool {
	signature := req.Signature
	req.Signature = nil
	data, _ := json.Marshal(req)
	req.Signature = signature

	return VerifySignature(pubKey, data, signature)
}

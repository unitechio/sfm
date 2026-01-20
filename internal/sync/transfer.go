package sync

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
	"github.com/owner/secure-file-manager/internal/crypto"
	"github.com/owner/secure-file-manager/internal/storage"
	"github.com/owner/secure-file-manager/pkg/models"
)

const (
	TransferProtocolID = "/sfm/transfer/1.0.0"
	ChunkSize          = 4 * 1024 * 1024 // 4MB
)

type TransferManager struct {
	node        *P2PNode
	onProgress  func(transferred, total int64)
	downloadDir string
}

func NewTransferManager(node *P2PNode, downloadDir string) *TransferManager {
	return &TransferManager{
		node:        node,
		downloadDir: downloadDir,
	}
}

// SetProgressCallback sets the progress callback
func (tm *TransferManager) SetProgressCallback(callback func(transferred, total int64)) {
	tm.onProgress = callback
}

// RegisterHandler registers the transfer protocol handler
func (tm *TransferManager) RegisterHandler() {
	tm.node.host.SetStreamHandler(protocol.ID(TransferProtocolID), tm.handleIncomingTransfer)
}

// SendFile sends a file to a peer
func (tm *TransferManager) SendFile(ctx context.Context, peerID peer.ID, filePath string) error {
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

	// Create stream to peer
	stream, err := tm.node.host.NewStream(ctx, peerID, protocol.ID(TransferProtocolID))
	if err != nil {
		return fmt.Errorf("failed to create stream: %w", err)
	}
	defer stream.Close()

	writer := bufio.NewWriter(stream)

	// Send metadata: filename length, filename, file size
	filename := filepath.Base(filePath)
	if err := binary.Write(writer, binary.LittleEndian, uint32(len(filename))); err != nil {
		return err
	}
	if _, err := writer.WriteString(filename); err != nil {
		return err
	}
	if err := binary.Write(writer, binary.LittleEndian, fileInfo.Size()); err != nil {
		return err
	}

	// Encrypt and send file
	key := make([]byte, 32)
	// In production, derive key from shared secret
	if _, err := io.ReadFull(file, key); err != nil && err != io.EOF {
		// For now, use a simple key derivation
		copy(key, []byte("temporary-key-for-demo-purposes"))
	}
	file.Seek(0, 0)

	// Send file in chunks
	transferred := int64(0)
	buffer := make([]byte, ChunkSize)
	hasher := sha256.New()

	for {
		n, err := file.Read(buffer)
		if err != nil && err != io.EOF {
			return fmt.Errorf("failed to read file: %w", err)
		}
		if n == 0 {
			break
		}

		// Encrypt chunk
		encrypted, err := crypto.Encrypt(buffer[:n], key)
		if err != nil {
			return fmt.Errorf("failed to encrypt chunk: %w", err)
		}

		// Send chunk size and data
		if err := binary.Write(writer, binary.LittleEndian, uint32(len(encrypted))); err != nil {
			return err
		}
		if _, err := writer.Write(encrypted); err != nil {
			return err
		}

		hasher.Write(buffer[:n])
		transferred += int64(n)

		if tm.onProgress != nil {
			tm.onProgress(transferred, fileInfo.Size())
		}
	}

	// Send checksum
	checksum := hasher.Sum(nil)
	if _, err := writer.Write(checksum); err != nil {
		return err
	}

	if err := writer.Flush(); err != nil {
		return err
	}

	// Record transfer
	tm.recordTransfer(peerID.String(), filePath, fileInfo.Size(), "send", "completed")

	return nil
}

func (tm *TransferManager) handleIncomingTransfer(stream network.Stream) {
	defer stream.Close()

	reader := bufio.NewReader(stream)

	// Read metadata
	var filenameLen uint32
	if err := binary.Read(reader, binary.LittleEndian, &filenameLen); err != nil {
		return
	}

	filenameBytes := make([]byte, filenameLen)
	if _, err := io.ReadFull(reader, filenameBytes); err != nil {
		return
	}
	filename := string(filenameBytes)

	var fileSize int64
	if err := binary.Read(reader, binary.LittleEndian, &fileSize); err != nil {
		return
	}

	// Create output file
	outputPath := filepath.Join(tm.downloadDir, filename)
	if err := os.MkdirAll(tm.downloadDir, 0755); err != nil {
		return
	}

	outFile, err := os.Create(outputPath)
	if err != nil {
		return
	}
	defer outFile.Close()

	// Receive and decrypt file
	key := make([]byte, 32)
	copy(key, []byte("temporary-key-for-demo-purposes"))

	received := int64(0)
	hasher := sha256.New()

	for received < fileSize {
		var chunkSize uint32
		if err := binary.Read(reader, binary.LittleEndian, &chunkSize); err != nil {
			return
		}

		encryptedChunk := make([]byte, chunkSize)
		if _, err := io.ReadFull(reader, encryptedChunk); err != nil {
			return
		}

		// Decrypt chunk
		decrypted, err := crypto.Decrypt(encryptedChunk, key)
		if err != nil {
			return
		}

		if _, err := outFile.Write(decrypted); err != nil {
			return
		}

		hasher.Write(decrypted)
		received += int64(len(decrypted))

		if tm.onProgress != nil {
			tm.onProgress(received, fileSize)
		}
	}

	// Verify checksum
	expectedChecksum := make([]byte, 32)
	if _, err := io.ReadFull(reader, expectedChecksum); err != nil {
		return
	}

	actualChecksum := hasher.Sum(nil)
	if string(expectedChecksum) != string(actualChecksum) {
		os.Remove(outputPath)
		return
	}

	// Record transfer
	peerID := stream.Conn().RemotePeer().String()
	tm.recordTransfer(peerID, outputPath, fileSize, "receive", "completed")
}

func (tm *TransferManager) recordTransfer(peerID, filePath string, fileSize int64, direction, status string) {
	db := storage.DB()

	// Get device name
	var device models.PairedDevice
	deviceName := "Unknown"
	if err := db.Where("peer_id = ?", peerID).First(&device).Error; err == nil {
		deviceName = device.DeviceName
	}

	transfer := models.TransferHistory{
		PeerID:     peerID,
		DeviceName: deviceName,
		FilePath:   filePath,
		FileSize:   fileSize,
		Status:     status,
		Direction:  direction,
		Progress:   100.0,
	}

	db.Create(&transfer)
}

// GetTransferHistory returns transfer history
func (tm *TransferManager) GetTransferHistory(limit int) ([]models.TransferHistory, error) {
	db := storage.DB()
	var history []models.TransferHistory
	err := db.Order("created_at DESC").Limit(limit).Find(&history).Error
	return history, err
}

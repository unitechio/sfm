package sync

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"math/big"
	"time"

	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/owner/secure-file-manager/internal/storage"
	"github.com/owner/secure-file-manager/pkg/models"
	"github.com/skip2/go-qrcode"
)

type PairingManager struct {
	node *P2PNode
}

func NewPairingManager(node *P2PNode) *PairingManager {
	return &PairingManager{node: node}
}

// GeneratePairingCode generates a pairing code and QR code
func (pm *PairingManager) GeneratePairingCode() (string, []byte, error) {
	// Generate 8-digit PIN
	pin, err := generatePIN(8)
	if err != nil {
		return "", nil, err
	}

	// Get peer ID
	peerID := pm.node.GetPeerID().String()

	// Get addresses
	addrs := pm.node.GetAddresses()
	addrStr := ""
	if len(addrs) > 0 {
		addrStr = addrs[0].String()
	}

	// Create pairing data: PIN|PeerID|Address
	pairingData := fmt.Sprintf("%s|%s|%s", pin, peerID, addrStr)

	// Generate QR code
	qrCode, err := qrcode.Encode(pairingData, qrcode.Medium, 256)
	if err != nil {
		return "", nil, fmt.Errorf("failed to generate QR code: %w", err)
	}

	return pairingData, qrCode, nil
}

// PairWithCode pairs with another device using a pairing code
func (pm *PairingManager) PairWithCode(ctx context.Context, pairingCode, deviceName string) error {
	// Parse pairing code: PIN|PeerID|Address
	var pin, peerIDStr, addrStr string
	fmt.Sscanf(pairingCode, "%s|%s|%s", &pin, &peerIDStr, &addrStr)

	// Parse peer ID
	peerID, err := peer.Decode(peerIDStr)
	if err != nil {
		return fmt.Errorf("invalid peer ID: %w", err)
	}

	// Connect to peer
	// In real implementation, would use the address to connect
	// For now, we'll rely on DHT discovery

	// Generate shared account ID (hash of both peer IDs)
	accountID := generateAccountID(pm.node.GetPeerID(), peerID)

	// Get peer's public key (would exchange via libp2p stream)
	pubKey, err := peerID.ExtractPublicKey()
	if err != nil {
		return fmt.Errorf("failed to extract public key: %w", err)
	}

	pubKeyBytes, err := crypto.MarshalPublicKey(pubKey)
	if err != nil {
		return fmt.Errorf("failed to marshal public key: %w", err)
	}

	// Save paired device
	db := storage.DB()
	pairedDevice := models.PairedDevice{
		PeerID:     peerIDStr,
		DeviceName: deviceName,
		PublicKey:  pubKeyBytes,
		AccountID:  accountID,
		LastSeen:   time.Now(),
		IsOnline:   true,
	}

	if err := db.Create(&pairedDevice).Error; err != nil {
		return fmt.Errorf("failed to save paired device: %w", err)
	}

	// Update local account info
	pm.updateAccountInfo(accountID)

	return nil
}

// ListPairedDevices returns all paired devices
func (pm *PairingManager) ListPairedDevices() ([]models.PairedDevice, error) {
	db := storage.DB()
	var devices []models.PairedDevice
	if err := db.Find(&devices).Error; err != nil {
		return nil, err
	}
	return devices, nil
}

// RevokePairing removes a paired device
func (pm *PairingManager) RevokePairing(peerID string) error {
	db := storage.DB()
	return db.Where("peer_id = ?", peerID).Delete(&models.PairedDevice{}).Error
}

func (pm *PairingManager) updateAccountInfo(accountID string) error {
	db := storage.DB()

	// Get or create account info
	var accountInfo models.AccountInfo
	result := db.Where("peer_id = ?", pm.node.GetPeerID().String()).FirstOrCreate(&accountInfo)
	if result.Error != nil {
		return result.Error
	}

	// Update account ID
	accountInfo.AccountID = accountID
	return db.Save(&accountInfo).Error
}

func generatePIN(length int) (string, error) {
	const digits = "0123456789"
	pin := make([]byte, length)
	for i := range pin {
		num, err := rand.Int(rand.Reader, big.NewInt(int64(len(digits))))
		if err != nil {
			return "", err
		}
		pin[i] = digits[num.Int64()]
	}
	return string(pin), nil
}

func generateAccountID(peer1, peer2 peer.ID) string {
	// Simple implementation: concatenate and encode
	combined := peer1.String() + peer2.String()
	return base64.StdEncoding.EncodeToString([]byte(combined))[:32]
}

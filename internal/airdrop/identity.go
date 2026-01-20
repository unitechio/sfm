package airdrop

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
)

// DeviceIdentity represents the device's cryptographic identity
type DeviceIdentity struct {
	PublicKey   ed25519.PublicKey
	PrivateKey  ed25519.PrivateKey
	Fingerprint string
}

// GenerateIdentity creates a new device identity
func GenerateIdentity() (*DeviceIdentity, error) {
	pubKey, privKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("failed to generate keypair: %w", err)
	}

	fingerprint := generateFingerprint(pubKey)

	return &DeviceIdentity{
		PublicKey:   pubKey,
		PrivateKey:  privKey,
		Fingerprint: fingerprint,
	}, nil
}

// LoadOrGenerateIdentity loads existing identity or generates new one
func LoadOrGenerateIdentity(dataDir string) (*DeviceIdentity, error) {
	pubKeyPath := filepath.Join(dataDir, "device.pub")
	privKeyPath := filepath.Join(dataDir, "device.key")

	// Try to load existing keys
	if pubKeyData, err := os.ReadFile(pubKeyPath); err == nil {
		if privKeyData, err := os.ReadFile(privKeyPath); err == nil {
			fingerprint := generateFingerprint(ed25519.PublicKey(pubKeyData))
			return &DeviceIdentity{
				PublicKey:   ed25519.PublicKey(pubKeyData),
				PrivateKey:  ed25519.PrivateKey(privKeyData),
				Fingerprint: fingerprint,
			}, nil
		}
	}

	// Generate new identity
	identity, err := GenerateIdentity()
	if err != nil {
		return nil, err
	}

	// Save keys
	if err := os.MkdirAll(dataDir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %w", err)
	}

	if err := os.WriteFile(pubKeyPath, identity.PublicKey, 0644); err != nil {
		return nil, fmt.Errorf("failed to save public key: %w", err)
	}

	if err := os.WriteFile(privKeyPath, identity.PrivateKey, 0600); err != nil {
		return nil, fmt.Errorf("failed to save private key: %w", err)
	}

	return identity, nil
}

// generateFingerprint creates a human-readable fingerprint from public key
func generateFingerprint(pubKey ed25519.PublicKey) string {
	hash := sha256.Sum256(pubKey)
	fingerprint := hex.EncodeToString(hash[:])

	// Format as XX:XX:XX:XX:...
	formatted := ""
	for i := 0; i < len(fingerprint); i += 2 {
		if i > 0 {
			formatted += ":"
		}
		formatted += fingerprint[i : i+2]
		if i >= 30 { // First 16 bytes only
			break
		}
	}
	return formatted
}

// Sign signs data with the private key
func (id *DeviceIdentity) Sign(data []byte) []byte {
	return ed25519.Sign(id.PrivateKey, data)
}

// Verify verifies a signature
func VerifySignature(pubKey ed25519.PublicKey, data, signature []byte) bool {
	return ed25519.Verify(pubKey, data, signature)
}

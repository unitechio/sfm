package models

import (
	"time"

	"gorm.io/gorm"
)

// EncryptedContainer represents an encrypted file/folder container
type EncryptedContainer struct {
	ID           uint           `gorm:"primarykey"`
	CreatedAt    time.Time
	UpdatedAt    time.Time
	DeletedAt    gorm.DeletedAt `gorm:"index"`
	Path         string         `gorm:"uniqueIndex;not null"`
	OriginalPath string         `gorm:"not null"`
	Salt         []byte         `gorm:"not null"`
	Argon2Time   uint32         `gorm:"not null"`
	Argon2Memory uint32         `gorm:"not null"`
	Argon2Threads uint8         `gorm:"not null"`
	IsMounted    bool           `gorm:"default:false"`
	MountPoint   string
}

// PairedDevice represents a device paired for P2P sync
type PairedDevice struct {
	ID           uint           `gorm:"primarykey"`
	CreatedAt    time.Time
	UpdatedAt    time.Time
	DeletedAt    gorm.DeletedAt `gorm:"index"`
	PeerID       string         `gorm:"uniqueIndex;not null"`
	DeviceName   string         `gorm:"not null"`
	PublicKey    []byte         `gorm:"not null"`
	AccountID    string         `gorm:"index;not null"`
	LastSeen     time.Time
	IsOnline     bool           `gorm:"default:false"`
	LocalAddress string
}

// TransferHistory tracks file transfer history
type TransferHistory struct {
	ID         uint           `gorm:"primarykey"`
	CreatedAt  time.Time
	UpdatedAt  time.Time
	DeletedAt  gorm.DeletedAt `gorm:"index"`
	PeerID     string         `gorm:"index;not null"`
	DeviceName string
	FilePath   string         `gorm:"not null"`
	FileSize   int64
	Status     string         `gorm:"not null"` // pending, transferring, completed, failed
	Direction  string         `gorm:"not null"` // send, receive
	Progress   float64        `gorm:"default:0"`
	Error      string
}

// AccountInfo stores local account information
type AccountInfo struct {
	ID         uint      `gorm:"primarykey"`
	CreatedAt  time.Time
	UpdatedAt  time.Time
	AccountID  string    `gorm:"uniqueIndex;not null"`
	DeviceName string    `gorm:"not null"`
	PeerID     string    `gorm:"not null"`
	PrivateKey []byte    `gorm:"not null"`
	PublicKey  []byte    `gorm:"not null"`
}

// SearchIndex represents indexed file metadata
type SearchIndex struct {
	ID           uint           `gorm:"primarykey"`
	CreatedAt    time.Time
	UpdatedAt    time.Time
	DeletedAt    gorm.DeletedAt `gorm:"index"`
	Path         string         `gorm:"uniqueIndex;not null"`
	FileName     string         `gorm:"index;not null"`
	FileSize     int64
	ModifiedTime time.Time
	IsDirectory  bool
	ContentHash  string
}

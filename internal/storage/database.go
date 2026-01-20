package storage

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/owner/secure-file-manager/pkg/models"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var db *gorm.DB

// Init initializes the database connection
func Init(dbPath string) error {
	// Ensure directory exists
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create database directory: %w", err)
	}

	var err error
	db, err = gorm.Open(sqlite.Open(dbPath), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}

	// Auto-migrate models
	if err := db.AutoMigrate(
		&models.EncryptedContainer{},
		&models.PairedDevice{},
		&models.TransferHistory{},
		&models.AccountInfo{},
		&models.SearchIndex{},
	); err != nil {
		return fmt.Errorf("failed to migrate database: %w", err)
	}

	return nil
}

// DB returns the database instance
func DB() *gorm.DB {
	if db == nil {
		panic("database not initialized, call Init() first")
	}
	return db
}

// Close closes the database connection
func Close() error {
	if db == nil {
		return nil
	}
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

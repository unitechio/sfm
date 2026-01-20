package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

type Config struct {
	Database DatabaseConfig `mapstructure:"database"`
	Crypto   CryptoConfig   `mapstructure:"crypto"`
	Search   SearchConfig   `mapstructure:"search"`
	Sync     SyncConfig     `mapstructure:"sync"`
	Logging  LoggingConfig  `mapstructure:"logging"`
}

type DatabaseConfig struct {
	Path string `mapstructure:"path"`
}

type CryptoConfig struct {
	Argon2Time    uint32 `mapstructure:"argon2_time"`
	Argon2Memory  uint32 `mapstructure:"argon2_memory"`
	Argon2Threads uint8  `mapstructure:"argon2_threads"`
	KeyLength     uint32 `mapstructure:"key_length"`
}

type SearchConfig struct {
	IndexPath      string `mapstructure:"index_path"`
	MaxWorkers     int    `mapstructure:"max_workers"`
	IndexContent   bool   `mapstructure:"index_content"`
	MaxContentSize int64  `mapstructure:"max_content_size"`
}

type SyncConfig struct {
	ListenPort     int      `mapstructure:"listen_port"`
	BootstrapPeers []string `mapstructure:"bootstrap_peers"`
	EnableMDNS     bool     `mapstructure:"enable_mdns"`
	RelayEnabled   bool     `mapstructure:"relay_enabled"`
	DataDir        string   `mapstructure:"data_dir"`
}

type LoggingConfig struct {
	Level  string `mapstructure:"level"`
	Output string `mapstructure:"output"`
}

var globalConfig *Config

// Load loads configuration from file or creates default
func Load() (*Config, error) {
	if globalConfig != nil {
		return globalConfig, nil
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	configDir := filepath.Join(homeDir, ".sfm")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create config directory: %w", err)
	}

	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(configDir)
	viper.AddConfigPath(".")

	// Set defaults
	setDefaults(configDir)

	// Read config file if exists
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			// Config file not found, create default
			if err := viper.SafeWriteConfig(); err != nil {
				return nil, fmt.Errorf("failed to write default config: %w", err)
			}
		} else {
			return nil, fmt.Errorf("failed to read config: %w", err)
		}
	}

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	globalConfig = &cfg
	return globalConfig, nil
}

func setDefaults(configDir string) {
	// Database
	viper.SetDefault("database.path", filepath.Join(configDir, "sfm.db"))

	// Crypto
	viper.SetDefault("crypto.argon2_time", 3)
	viper.SetDefault("crypto.argon2_memory", 65536) // 64MB
	viper.SetDefault("crypto.argon2_threads", 4)
	viper.SetDefault("crypto.key_length", 32)

	// Search
	viper.SetDefault("search.index_path", filepath.Join(configDir, "search_index"))
	viper.SetDefault("search.max_workers", 8)
	viper.SetDefault("search.index_content", true)
	viper.SetDefault("search.max_content_size", 10*1024*1024) // 10MB

	// Sync
	viper.SetDefault("sync.listen_port", 0) // Random port
	viper.SetDefault("sync.bootstrap_peers", []string{
		"/dnsaddr/bootstrap.libp2p.io/p2p/QmNnooDu7bfjPFoTZYxMNLWUQJyrVwtbZg5gBMjTezGAJN",
		"/dnsaddr/bootstrap.libp2p.io/p2p/QmQCU2EcMqAqQPR2i9bChDtGNJchTbq5TbXJJ16u19uLTa",
	})
	viper.SetDefault("sync.enable_mdns", true)
	viper.SetDefault("sync.relay_enabled", true)
	viper.SetDefault("sync.data_dir", filepath.Join(configDir, "p2p"))

	// Logging
	viper.SetDefault("logging.level", "info")
	viper.SetDefault("logging.output", filepath.Join(configDir, "sfm.log"))
}

// Get returns the global config instance
func Get() *Config {
	if globalConfig == nil {
		panic("config not loaded, call Load() first")
	}
	return globalConfig
}

package config

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

// Config encapsulates runtime operational settings for auditforge
type Config struct {
	HTTPPort      int           `yaml:"http_port"`
	GRPCPort      int           `yaml:"grpc_port"`
	DataDir       string        `yaml:"data_dir"`
	WALPath       string        `yaml:"wal_path"`
	DBPath        string        `yaml:"db_path"`
	KeyPath       string        `yaml:"key_path"`
	Algorithm     string        `yaml:"algorithm"`
	BatchSize     int           `yaml:"batch_size"`
	FlushInterval time.Duration `yaml:"flush_interval"`
	APIKeys       []string      `yaml:"api_keys"`
}

// DefaultConfig returns ready-to-run local development defaults
func DefaultConfig() *Config {
	dataDir := "./data"
	return &Config{
		HTTPPort:      8080,
		GRPCPort:      9090,
		DataDir:       dataDir,
		WALPath:       filepath.Join(dataDir, "ledger.wal"),
		DBPath:        filepath.Join(dataDir, "index.db"),
		KeyPath:       filepath.Join(dataDir, "signer.key"),
		Algorithm:     "blake3",
		BatchSize:     500,
		FlushInterval: 100 * time.Millisecond,
		APIKeys:       []string{"default-dev-key"},
	}
}

// LoadConfig parses a YAML configuration file or falls back to defaults
func LoadConfig(path string) (*Config, error) {
	cfg := DefaultConfig()

	if path == "" {
		return cfg, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return nil, fmt.Errorf("failed reading config file %s: %w", path, err)
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("failed parsing config yaml: %w", err)
	}

	return cfg, nil
}

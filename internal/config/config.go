package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server  ServerConfig  `yaml:"server"`
	Rpc     RpcConfig     `yaml:"rpc"`
	DB      DBConfig      `yaml:"db"`
	Indexer IndexerConfig `yaml:"indexer"`
}

type ServerConfig struct {
	Port string `yaml:"port"`
}

type RpcConfig struct {
	ChainID        int64  `yaml:"chain_id"`
	RPCURL         string `yaml:"rpc_url"`
	TimeoutSeconds int    `yaml:"timeout_seconds"`
}

type DBConfig struct {
	DSN string `yaml:"dsn"`
}

type IndexerConfig struct {
	AutoStart         bool   `yaml:"auto_start"`
	IntervalSeconds   int    `yaml:"interval_seconds"`
	RunTimeoutSeconds int    `yaml:"run_timeout_seconds"`
	SyncTarget        string `yaml:"sync_target"`
	StartBlock        uint64 `yaml:"start_block"`
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("unmarshal yaml: %w", err)
	}

	if cfg.Server.Port == "" {
		cfg.Server.Port = "8080"
	}

	if cfg.Rpc.ChainID <= 0 {
		return nil, fmt.Errorf("rpc.chain_id must be positive")
	}
	if cfg.Rpc.RPCURL == "" {
		return nil, fmt.Errorf("rpc.rpc_url is required")
	}
	if cfg.Rpc.TimeoutSeconds <= 0 {
		cfg.Rpc.TimeoutSeconds = 10
	}

	if cfg.DB.DSN == "" {
		return nil, fmt.Errorf("db.dsn is required")
	}

	if cfg.Indexer.IntervalSeconds <= 0 {
		cfg.Indexer.IntervalSeconds = 2
	}

	if cfg.Indexer.RunTimeoutSeconds <= 0 {
		cfg.Indexer.RunTimeoutSeconds = 3
	}
	if cfg.Indexer.SyncTarget == "" {
		cfg.Indexer.SyncTarget = "safe"
	}
	if cfg.Indexer.SyncTarget != "latest" &&
		cfg.Indexer.SyncTarget != "safe" &&
		cfg.Indexer.SyncTarget != "finalized" {
		return nil, fmt.Errorf("invalid indexer sync_target: %s", cfg.Indexer.SyncTarget)
	}

	return &cfg, nil
}

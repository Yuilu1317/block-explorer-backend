package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server ServerConfig `yaml:"server"`
	Rpc    RpcConfig    `yaml:"rpc"`
	DB     DBConfig     `yaml:"db"`
}

type ServerConfig struct {
	Port string `yaml:"port"`
}

type RpcConfig struct {
	RPCURL         string `yaml:"rpc_url"`
	TimeoutSeconds int    `yaml:"timeout_seconds"`
}

type DBConfig struct {
	DSN string `yaml:"dsn"`
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
	if cfg.Rpc.RPCURL == "" {
		return nil, fmt.Errorf("rpc.rpc_url is required")
	}
	if cfg.Rpc.TimeoutSeconds <= 0 {
		cfg.Rpc.TimeoutSeconds = 10
	}

	if cfg.DB.DSN == "" {
		return nil, fmt.Errorf("db.dsn is required")
	}

	return &cfg, nil
}

package config

import (
	"encoding/json"
	"fmt"
	"os"
)

func LoadServerConfig(path string) (ServerConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return ServerConfig{}, fmt.Errorf("read config: %w", err)
	}

	var config ServerConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return ServerConfig{}, fmt.Errorf("parse config: %w", err)
	}

	return config, nil
}

func LoadClientConfig(path string) (ClientConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return ClientConfig{}, fmt.Errorf("read config: %w", err)
	}

	var config ClientConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return ClientConfig{}, fmt.Errorf("parse config: %w", err)
	}

	return config, nil
}

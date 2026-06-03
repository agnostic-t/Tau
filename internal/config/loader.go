package config

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"strings"
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

func EncodeServerConfig(ip string, inbound ServerInbound) (string, error) {
	clientServer := ClientsServer{
		Address:   fmt.Sprintf("%s:%d", ip, inbound.Port),
		Obfs:      inbound.Obfs,
		Handshake: inbound.Handshake,
		Trans:     inbound.Trans,
		Mux:       inbound.Mux,
	}

	data, err := json.Marshal(clientServer)
	if err != nil {
		return "", fmt.Errorf("failed to marshal client server config: %w", err)
	}

	encoded := base64.StdEncoding.EncodeToString(data)
	return "tau://" + encoded, nil
}

func DecodeClientConfig(encoded string) (*ClientsServer, error) {
	if !strings.HasPrefix(encoded, "tau://") {
		return nil, fmt.Errorf("config string has no tau://, can be different proto")
	}

	encoded, _ = strings.CutPrefix(encoded, "tau://")

	data, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, fmt.Errorf("failed to decode base64: %w", err)
	}

	var clientServer ClientsServer
	if err := json.Unmarshal(data, &clientServer); err != nil {
		return nil, fmt.Errorf("failed to unmarshal client server config: %w", err)
	}

	return &clientServer, nil
}

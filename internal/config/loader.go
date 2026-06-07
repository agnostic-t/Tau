package config

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

type configWrapper struct {
	Name string        `json:"name"`
	Inb  ClientsServer `json:"inb"`
}

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

func EncodeServerConfig(name, ip string, inbound ServerInbound) (string, error) {
	clientServer := ClientsServer{
		Address:   fmt.Sprintf("%s:%d", ip, inbound.Port),
		Obfs:      inbound.Obfs,
		Handshake: inbound.Handshake,
		Trans:     inbound.Trans,
		Mux:       inbound.Mux,
	}

	wrapper := configWrapper{
		Name: name,
		Inb:  clientServer,
	}

	data, err := json.Marshal(wrapper)
	if err != nil {
		return "", fmt.Errorf("failed to marshal client server config: %w", err)
	}

	encoded := base64.StdEncoding.EncodeToString(data)
	return "tau://" + encoded, nil
}

func DecodeClientConfig(encoded string) (*ClientsServer, string, error) {
	if !strings.HasPrefix(encoded, "tau://") {
		return nil, "", fmt.Errorf("config string has no tau://, can be different proto")
	}

	encoded, _ = strings.CutPrefix(encoded, "tau://")

	data, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, "", fmt.Errorf("failed to decode base64: %w", err)
	}

	var clientServerWrapped configWrapper
	if err := json.Unmarshal(data, &clientServerWrapped); err != nil {
		return nil, "", fmt.Errorf("failed to unmarshal client server config: %w", err)
	}

	return &clientServerWrapped.Inb, clientServerWrapped.Name, nil
}

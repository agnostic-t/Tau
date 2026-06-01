package config

import (
	"encoding/json"
	"errors"
	"fmt"
)

type ModuleConfig struct {
	Type     string         `json:"type"`
	Settings map[string]any `json:"settings,omitempty"`
}

func (m *ModuleConfig) UnmarshalJSON(data []byte) error {
	var obj struct {
		Type     string         `json:"type"`
		Settings map[string]any `json:"settings,omitempty"`
	}
	if err := json.Unmarshal(data, &obj); err == nil && obj.Type != "" {
		m.Type = obj.Type
		m.Settings = obj.Settings
		return nil
	}

	var str string
	if err := json.Unmarshal(data, &str); err == nil {
		m.Type = str
		return nil
	}

	return fmt.Errorf("invalid module config: expected string or object with 'type' field")
}

func (m *ModuleConfig) GetSetting(key string) (any, bool) {
	if m.Settings == nil {
		return nil, false
	}
	val, ok := m.Settings[key]
	return val, ok
}

func (m *ModuleConfig) DecodeSettings(v any) error {
	if m.Settings == nil {
		return nil
	}
	data, err := json.Marshal(m.Settings)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, v)
}

type ServerConfig struct {
	BindIP      string                   `json:"bindIP"`
	ExternalIP  string                   `json:"externalIP"`
	Inbounds    map[string]ServerInbound `json:"inbounds"`
	LockedIPs   map[string]string        `json:"lockedIPs,omitempty"`
	UsedTraffic map[string]int           `json:"usedTraffic,omitempty"`
}

type ServerInbound struct {
	Port      int          `json:"port"`
	Obfs      ModuleConfig `json:"obfs"`
	Handshake ModuleConfig `json:"handshake"`
	Trans     ModuleConfig `json:"trans"`
	Mux       ModuleConfig `json:"mux"`
}

type ClientConfig struct {
	LProxy   map[string]string        `json:"lproxy"`
	Selected string                   `json:"selected"`
	Servers  map[string]ClientsServer `json:"servers"`
}

type ClientsServer struct {
	Address   string       `json:"addr"`
	Obfs      ModuleConfig `json:"obfs"`
	Handshake ModuleConfig `json:"handshake"`
	Trans     ModuleConfig `json:"trans"`
	Mux       ModuleConfig `json:"mux"`
	Traffic   string       `json:"traffic,omitempty"`
	Locked    bool         `json:"locked,omitempty"`
}

func (c *ClientConfig) Validate() error {
	if c.Selected == "" {
		return errors.New("no server selected")
	}
	if len(c.Servers) == 0 {
		return errors.New("no servers added")
	}
	if _, ok := c.Servers[c.Selected]; !ok {
		return errors.New("selected server is not present in config")
	}
	return nil
}

func (c *ServerConfig) Validate() error {
	if c.BindIP == "" {
		return errors.New("bindIP is required")
	}
	for id, inbound := range c.Inbounds {
		if inbound.Port <= 0 || inbound.Port > 65535 {
			return fmt.Errorf("inbound %s: invalid port %d", id, inbound.Port)
		}
	}
	return nil
}

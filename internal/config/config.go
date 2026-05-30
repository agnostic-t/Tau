package config

import (
	"errors"
	"fmt"
)

type ServerConfig struct {
	BindIP     string `json:"bindIP"`
	ExternalIP string `json:"externalIP"`

	Inbounds map[string]ServerInbound `json:"inbounds"`
	Clients  map[string]ServersClient `json:"clients"`

	LockedIPs   map[string]string `json:"lockedIPs,omitempty"`
	UsedTraffic map[string]int    `json:"usedTraffic,omitempty"`
}

type ServerInbound struct {
	Psk    string `json:"psk"`
	Port   int    `json:"port"`
	Obfs   string `json:"obfs"`
	Trans  string `json:"trans"`
	Handsh string `json:"handshake"`
}

type ServersClient struct {
	Traffic string `json:"traffic"`
	Locked  bool   `json:"locked"`
	Inbound string `json:"inbound"`
}

type ClientConfig struct {
	Lproxy   map[string]string        `json:"lproxy"`
	Selected string                   `json:"selected"`
	Servers  map[string]ClientsServer `json:"servers"`
}

type ClientsServer struct {
	Address string `json:"addr"`
	Obfs    string `json:"obfs"`
	Psk     string `json:"psk"`
	Traffic string `json:"traffic"`
	Locked  bool   `json:"locked"`
	Handsh  string `json:"handshake"`
}

func (c *ClientConfig) Validate() error {
	if c.Selected == "" {
		return errors.New("No server selected")
	}

	if len(c.Servers) == 0 {
		return errors.New("No servers added")
	}

	if _, ok := c.Servers[c.Selected]; !ok {
		return errors.New("Selected server is not presented in config")
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
	for id, client := range c.Clients {
		if _, ok := c.Inbounds[client.Inbound]; !ok {
			return fmt.Errorf("client %s: unknown inbound %s", id, client.Inbound)
		}
	}
	return nil
}

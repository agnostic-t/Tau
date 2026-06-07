package config

type ObfsTypeXOBFS struct {
	Psk string `json:"psk"`
}

type HandshakeTypeXOBFS struct {
	Psk             string `json:"psk"`
	StartJunk       bool   `json:"startJunk"`
	RotateSeconds   int    `json:"rotateSeconds"`
	RotateJunkCount bool   `json:"rotateJunkCount"`
	MinJunkPacks    int    `json:"minJunkPacks"`
	MaxJunkPacks    int    `json:"maxJunkPacks"`
}

type TransportTypeHTTP struct {
	UserAgent string `json:"userAgent"`
	Referer   string `json:"referer"`
	KeyPath   string `json:"keyPath"`
}

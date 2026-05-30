package main

import (
	"context"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	b64 "encoding/base64"
	"encoding/json"

	"github.com/agnostic-t/neutrino-core/core/client"
	"github.com/agnostic-t/neutrino-core/handshake"
	"github.com/agnostic-t/neutrino-core/local"
	"github.com/agnostic-t/neutrino-core/obfuscation"
	"github.com/agnostic-t/neutrino-core/transport"
	"github.com/agnostic-t/neutrino-handsh/handshake/basic"
	"github.com/agnostic-t/neutrino-obfs/xobfs"
	"github.com/agnostic-t/neutrino-vpn/internal/config"

	"github.com/agnostic-t/neutrino-lproxies/socks5"
	"github.com/agnostic-t/neutrino-transport/basic/tcp"
)

func convertB64ToInbound(inb string) (config.ClientsServer, error) {
	jsonBytes, err := b64.StdEncoding.DecodeString(inb)
	if err != nil {
		return config.ClientsServer{}, err
	}

	var config config.ClientsServer
	if err := json.Unmarshal(jsonBytes, &config); err != nil {
		return config, err
	}

	return config, nil
}

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	if len(os.Args) == 1 {
		logger.Error("Failed to get config path, use program as: " + os.Args[0] + " PATH_TO_CONFIG")
		os.Exit(-1)
	}

	pathConfig := os.Args[1]

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	config, err := config.LoadClientConfig(pathConfig)
	if err != nil {
		logger.Error("Failed to load config", "error", err)
		os.Exit(-1)
	}

	proxies := make(map[string]local.Proxy)
	defer func() {
		for _, prx := range proxies {
			prx.Close()
		}
	}()

	for prx_type, addr := range config.Lproxy {
		var proxy local.Proxy

		switch prx_type {
		case "socks5":
			proxy = socks5.NewProxy(addr)
		default:
			logger.Error("Unknown local proxy type", "type", prx_type)
			os.Exit(-1)
		}

		proxies[prx_type+"://"+addr] = proxy
	}

	server := config.Servers[config.Selected]
	trans := tcp.NewClient(server.Address, 5*time.Second)

	var obfs obfuscation.Obfuscator

	switch server.Obfs {
	case "xobfs":
		obfs = &xobfs.Obfuscator{Psk: []byte(server.Psk)}
	case "null":
		obfs = &NullObfuscator{}
	default:
		logger.Error("Invalid obfuscation algorithm", "type", server.Obfs)
		os.Exit(-1)
	}

	var handshake handshake.HandshakeHandler
	switch server.Handsh {
	case "plain":
		handshake = &basic.BasicHandshaker{}
	default:
		logger.Error("Invalid handshake algorithm", "type", server.Handsh)
		os.Exit(-1)
	}

	var wg sync.WaitGroup
	for addr, prx := range proxies {
		go startClient(addr, prx, handshake, trans, obfs, logger, ctx, &wg)
		wg.Add(1)
	}

	wg.Wait()
}

func startClient(
	laddr string,
	lproxy local.Proxy,
	handsh handshake.HandshakeHandler,
	trans transport.Client,
	obfs obfuscation.Obfuscator,
	logger *slog.Logger,
	ctx context.Context,
	wg *sync.WaitGroup,
) {
	client := client.NewClient(lproxy, trans, obfs, handsh, logger)

	logger.Info("Starting Neutrino Client", "laddr", laddr)

	if err := client.Start(ctx); err != nil {
		logger.Error("Failed to start client", "error", err)
	}

	wg.Done()
}

type NullObfuscator struct {
}

func (o *NullObfuscator) WrapConnTo(conn net.Conn) (net.Conn, error) {
	return conn, nil
}

func (o *NullObfuscator) WrapConnFrom(conn net.Conn) (net.Conn, error) {
	return conn, nil
}

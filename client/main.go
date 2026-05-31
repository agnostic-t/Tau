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

	"github.com/agnostic-t/neutrino-core/core/client"
	"github.com/agnostic-t/neutrino-core/handshake"
	"github.com/agnostic-t/neutrino-core/local"
	"github.com/agnostic-t/neutrino-core/obfuscation"
	"github.com/agnostic-t/neutrino-core/transport"
	"github.com/agnostic-t/neutrino-handsh/handshake/basic"
	"github.com/agnostic-t/neutrino-handsh/handshake/obfsh"
	"github.com/agnostic-t/neutrino-obfs/xobfs"
	iconf "github.com/agnostic-t/neutrino-vpn/internal/config"

	"github.com/agnostic-t/neutrino-lproxies/socks5"
	"github.com/agnostic-t/neutrino-transport/basic/tcp"
)

// func convertB64ToInbound(inb string) (config.ClientsServer, error) {
// 	jsonBytes, err := b64.StdEncoding.DecodeString(inb)
// 	if err != nil {
// 		return config.ClientsServer{}, err
// 	}

// 	var config config.ClientsServer
// 	if err := json.Unmarshal(jsonBytes, &config); err != nil {
// 		return config, err
// 	}

// 	return config, nil
// }

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	if len(os.Args) == 1 {
		logger.Error("Failed to get config path, use program as: " + os.Args[0] + " PATH_TO_CONFIG")
		os.Exit(-1)
	}

	pathConfig := os.Args[1]

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	config, err := iconf.LoadClientConfig(pathConfig)
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

	for prx_type, addr := range config.LProxy {
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

	switch server.Obfs.Type {
	case "xobfs":
		var opts iconf.ObfsTypeXOBFS
		if err := server.Obfs.DecodeSettings(&opts); err != nil {
			logger.Error("Failed to get opts for obfs", "type", server.Obfs.Type, "error", err)
			os.Exit(-1)
		}

		obfs = &xobfs.Obfuscator{Psk: []byte(opts.Psk)}
	case "null":
		obfs = &NullObfuscator{}
	default:
		logger.Error("Invalid obfuscation algorithm", "type", server.Obfs.Type)
		os.Exit(-1)
	}

	var handsh handshake.HandshakeHandler
	switch server.Handshake.Type {
	case "plain":
		handsh = &basic.BasicHandshaker{}
	case "xobfs":

		var opts iconf.HandshakeTypeXOBFS
		if err := server.Handshake.DecodeSettings(&opts); err != nil {
			logger.Error("Failed to get opts for handshake", "type", server.Handshake.Type, "error", err)
			os.Exit(-1)
		}

		handsh = obfsh.NewObfsHandshaker(
			[]byte(opts.Psk),
			opts.StartJunk,
			int64(opts.RotateSeconds),
			opts.RotateJunkCount,
			opts.MinJunkPacks,
			opts.MaxJunkPacks,
		)
	default:
		logger.Error("Invalid handshake algorithm", "type", server.Handshake.Type)
		os.Exit(-1)
	}

	var wg sync.WaitGroup
	for addr, prx := range proxies {
		go startClient(addr, prx, handsh, trans, obfs, logger, ctx, &wg)
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

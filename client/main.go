package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/agnostic-t/neutrino-core/core/client"
	"github.com/agnostic-t/neutrino-core/handshake"
	"github.com/agnostic-t/neutrino-core/local"
	"github.com/agnostic-t/neutrino-core/nmux"
	"github.com/agnostic-t/neutrino-core/obfuscation"
	"github.com/agnostic-t/neutrino-core/transport"
	"github.com/agnostic-t/neutrino-handsh/handshake/basic"
	"github.com/agnostic-t/neutrino-handsh/handshake/obfsh"
	"github.com/agnostic-t/neutrino-mux/yamuxed"
	"github.com/agnostic-t/neutrino-obfs/nobfs"
	"github.com/agnostic-t/neutrino-obfs/xobfs"
	iconf "github.com/agnostic-t/neutrino-vpn/internal/config"
	"github.com/agnostic-t/neutrino-vpn/internal/routing"
	itun "github.com/agnostic-t/neutrino-vpn/internal/tun"
	"github.com/hashicorp/yamux"

	"github.com/agnostic-t/neutrino-lproxies/socks5"
	"github.com/agnostic-t/neutrino-transport/basic/tcp"
)

func startTUN(
	tunIF string,
	mainIF string,
	gateway string,
	serverIP string,

	ctx context.Context,
	logger *slog.Logger,
) *itun.Manager {
	tunman, err := itun.NewManager(tunIF, mainIF, gateway, serverIP)
	if err != nil {
		logger.Error("Failed to start tun manager on", "tunIF", tunIF, "mainIF", mainIF, "gateway", gateway, "error", err)
		os.Exit(-1)
	}

	if err := tunman.Enable(); err != nil {
		logger.Error("Failed to enable tun manager", "error", err)
		os.Exit(-1)
	}

	go func() {
		<-ctx.Done()
		itun.StopTUN2SOCKS()
	}()

	return tunman
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
			proxy = socks5.NewProxy(addr, ctx)
		default:
			logger.Error("Unknown local proxy type", "type", prx_type)
			os.Exit(-1)
		}

		proxies[prx_type+"://"+addr] = proxy
	}

	server := config.Servers[config.Selected]
	trans := tcp.NewClient(server.Address, 5*time.Second)

	var tunman *itun.Manager = nil
	if config.Tun != nil && config.Tun.Enabled {
		logger.Info("Enabling TUN")
		tunman = startTUN(
			config.Tun.TunIF,
			config.Tun.MainIF,
			config.Tun.Gateway,
			strings.Split(server.Address, ":")[0],
			ctx,
			logger,
		)
	} else {
		logger.Info("TUN is disabled")
	}

	muxEnabled := true
	var muxer nmux.Multiplexer

	switch server.Mux.Type {
	case "null":
		muxEnabled = false
	case "yamux":
		cfg := yamux.DefaultConfig()
		cfg.EnableKeepAlive = true
		muxer = yamuxed.NewYamuxed(cfg)
	default:
		logger.Error("Invalid mux method", "type", server.Mux.Type)
		os.Exit(-1)
	}
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
		obfs = &nobfs.NullObfuscator{}
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

	var directIF string = ""
	if config.Tun != nil {
		directIF = config.Tun.MainIF
	}

	var flt local.Filter

	filteringEnabled := config.Filter != nil && config.Filter.Enabled
	if filteringEnabled {
		flt, err = routing.NewRoutingFilter(config.Filter.DirectPath, config.Filter.BlockPath)
		if err != nil {
			logger.Error("Failed to compile routing rules", "error", err)
			os.Exit(-1)
		}

	} else {
		flt = &routing.DummyFilter{}
	}

	var wg sync.WaitGroup
	for addr, prx := range proxies {
		if tunman != nil {
			itun.StartTUN2SOCKS(tunman, addr)
		}

		go startClient(
			addr,
			prx,
			handsh,
			trans,
			obfs,
			muxer,
			muxEnabled,
			flt,
			directIF,
			logger,
			ctx,
			&wg,
		)
		wg.Add(1)
	}

	wg.Wait()

	if tunman != nil {
		logger.Info("Disabling TUN interface...")
		if err := tunman.Disable(); err != nil {
			logger.Warn("Failed to properly disable TUN", "error", err)
		}
	}
}

func startClient(
	laddr string,
	lproxy local.Proxy,
	handsh handshake.HandshakeHandler,
	trans transport.Client,
	obfs obfuscation.Obfuscator,
	muxer nmux.Multiplexer,
	enabledMuxer bool,
	filter local.Filter,
	directIF string,
	logger *slog.Logger,
	ctx context.Context,
	wg *sync.WaitGroup,
) {
	client := client.NewClient(lproxy, trans, obfs, handsh, muxer, filter, directIF, enabledMuxer, logger)

	logger.Info("Starting Neutrino Client", "laddr", laddr)

	if err := client.Start(ctx); err != nil {
		logger.Error("Failed to start client", "error", err)
	}

	wg.Done()
}

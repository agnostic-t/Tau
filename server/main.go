package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"

	"github.com/agnostic-t/neutrino-core/core/server"
	"github.com/agnostic-t/neutrino-core/handshake"
	"github.com/agnostic-t/neutrino-core/nmux"
	"github.com/agnostic-t/neutrino-core/obfuscation"
	"github.com/agnostic-t/neutrino-core/transport"
	"github.com/agnostic-t/neutrino-mux/yamuxed"
	"github.com/hashicorp/yamux"

	"github.com/agnostic-t/neutrino-handsh/handshake/basic"
	"github.com/agnostic-t/neutrino-handsh/handshake/obfsh"
	"github.com/agnostic-t/neutrino-obfs/nobfs"
	"github.com/agnostic-t/neutrino-obfs/xobfs"
	"github.com/agnostic-t/neutrino-transport/basic/tcp"

	iconf "github.com/agnostic-t/neutrino-vpn/internal/config"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	if len(os.Args) == 1 {
		logger.Error("Failed to get config path, use program as: " + os.Args[0] + " PATH_TO_CONFIG")
		os.Exit(-1)
	}

	pathConfig := os.Args[1]

	config, err := iconf.LoadServerConfig(pathConfig)
	if err != nil {
		logger.Error("Failed to parse config", "error", err)
		os.Exit(-1)
	}

	if err := config.Validate(); err != nil {
		logger.Error("Invalid config", "error", err)
		os.Exit(-1)
	}

	var wg sync.WaitGroup
	for name, inb := range config.Inbounds {
		logger.Info("Processing inbound", "name", name)

		b64_str, err := iconf.EncodeServerConfig(config.ExternalIP, inb)
		if err != nil {
			logger.Error("Failed to get B64 string for inbound", "inb", inb)
			os.Exit(-1)
		}

		logger.Info("For this inbound B64: ", "b64", b64_str)

		var obfs obfuscation.Obfuscator
		var trans transport.Server
		var handsh handshake.HandshakeHandler

		switch inb.Obfs.Type {
		case "xobfs":
			var opts iconf.ObfsTypeXOBFS
			if err := inb.Obfs.DecodeSettings(&opts); err != nil {
				logger.Error("Failed to get opts for obfs", "inb", name, "name", inb.Obfs.Type, "error", err)
				os.Exit(-1)
			}

			obfs = &xobfs.Obfuscator{Psk: []byte(opts.Psk)}
		case "null":
			obfs = &nobfs.NullObfuscator{}
		default:
			logger.Error("Invalid obfuscation method", "inb", name, "name", inb.Obfs)
			os.Exit(-1)
		}

		switch inb.Trans.Type {
		case "tcp":
			trans = tcp.NewServer(config.BindIP + ":" + strconv.Itoa(inb.Port))
		default:
			logger.Error("Invalid transport method", "inb", name, "name", inb.Trans)
			os.Exit(-1)
		}

		switch inb.Handshake.Type {
		case "plain":
			handsh = &basic.BasicHandshaker{}
		case "xobfs":

			var opts iconf.HandshakeTypeXOBFS
			if err := inb.Handshake.DecodeSettings(&opts); err != nil {
				logger.Error("Failed to get opts for handshake", "inb", name, "type", inb.Handshake.Type, "error", err)
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
			logger.Error("Invalid handshake method", "inb", name, "type", inb.Handshake.Type)
			os.Exit(-1)
		}

		muxEnabled := true
		var muxer nmux.Multiplexer

		switch inb.Mux.Type {
		case "null":
			muxEnabled = false
		case "yamux":
			cfg := yamux.DefaultConfig()
			cfg.EnableKeepAlive = true
			muxer = yamuxed.NewYamuxed(cfg)
		default:
			logger.Error("Invalid mux method", "inb", name, "type", inb.Mux.Type)
			os.Exit(-1)
		}

		go startServer(
			config.BindIP+":"+strconv.Itoa(inb.Port),
			handsh,
			trans,
			obfs,
			muxer,
			muxEnabled,
			logger,
			&wg,
		)
		wg.Add(1)
	}

	wg.Wait()
}

func startServer(
	addr string,
	handsh handshake.HandshakeHandler,
	trans transport.Server,
	obfs obfuscation.Obfuscator,
	muxer nmux.Multiplexer,
	enabledMuxer bool,
	logger *slog.Logger,
	wg *sync.WaitGroup,
) {
	defer wg.Done()

	server := server.NewServer(trans, obfs, handsh, muxer, enabledMuxer, logger)

	logger.Info("Server is starting", "addr", addr)

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	if err := server.Start(ctx); err != nil {
		logger.Error("Failed to start server", "error", err)
	}
}

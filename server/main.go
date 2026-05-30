package main

import (
	"context"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"

	"github.com/agnostic-t/neutrino-core/core/server"
	"github.com/agnostic-t/neutrino-core/obfuscation"
	"github.com/agnostic-t/neutrino-core/transport"

	"github.com/agnostic-t/neutrino-obfs/xobfs"
	"github.com/agnostic-t/neutrino-transport/basic/tcp"

	"github.com/agnostic-t/neutrino-vpn/internal/config"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	config, err := config.LoadServerConfig("./config/server.json")
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
		var obfs obfuscation.Obfuscator
		var trans transport.Server

		switch inb.Obfs {
		case "xobfs":
			obfs = &xobfs.Obfuscator{Psk: []byte(inb.Psk)}
		case "null":
			obfs = &NullObfuscator{}
		default:
			logger.Error("Invalid obfuscation method", "inb", name, "name", inb.Obfs)
			os.Exit(-1)
		}

		switch inb.Trans {
		case "tcp":
			trans = tcp.NewServer(config.BindIP + ":" + strconv.Itoa(inb.Port))
		default:
			logger.Error("Invalid transport method", "inb", name, "name", inb.Trans)
		}

		go startServer(
			config.BindIP+":"+strconv.Itoa(inb.Port),
			trans,
			obfs,
			logger,
			&wg,
		)
		wg.Add(1)
	}

	wg.Wait()
}

func startServer(addr string, trans transport.Server, obfs obfuscation.Obfuscator, logger *slog.Logger, wg *sync.WaitGroup) {
	defer wg.Done()

	server := server.NewServer(trans, obfs, logger)

	logger.Info("Server is starting", "addr", addr)

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	if err := server.Start(ctx); err != nil {
		logger.Error("Failed to start server", "error", err)
	}
}

type NullObfuscator struct {
}

func (o *NullObfuscator) WrapConnTo(conn net.Conn) (net.Conn, error) {
	return conn, nil
}

func (o *NullObfuscator) WrapConnFrom(conn net.Conn) (net.Conn, error) {
	return conn, nil
}

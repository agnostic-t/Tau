package main

import (
	"context"
	"encoding/json"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"

	b64 "encoding/base64"

	"github.com/agnostic-t/neutrino-core/core/server"
	"github.com/agnostic-t/neutrino-core/handshake"
	"github.com/agnostic-t/neutrino-core/obfuscation"
	"github.com/agnostic-t/neutrino-core/transport"

	"github.com/agnostic-t/neutrino-handsh/handshake/basic"
	"github.com/agnostic-t/neutrino-obfs/xobfs"
	"github.com/agnostic-t/neutrino-transport/basic/tcp"

	"github.com/agnostic-t/neutrino-vpn/internal/config"
)

func exportInboundAsB64(externalIP string, inb config.ServerInbound) (string, error) {
	inbClient := config.ClientsServer{
		Address: externalIP + ":" + strconv.Itoa(inb.Port),
		Obfs:    inb.Obfs,
		Psk:     inb.Psk,
		Traffic: inb.Trans,
		Locked:  false,
		Handsh:  inb.Handsh,
	}

	jsonBytes, err := json.Marshal(&inbClient)
	if err != nil {
		return "", err
	}

	return b64.StdEncoding.EncodeToString(jsonBytes), nil
}

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	if len(os.Args) == 1 {
		logger.Error("Failed to get config path, use program as: " + os.Args[0] + " PATH_TO_CONFIG")
		os.Exit(-1)
	}

	pathConfig := os.Args[1]

	config, err := config.LoadServerConfig(pathConfig)
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

		b64_str, err := exportInboundAsB64(config.ExternalIP, inb)
		if err != nil {
			logger.Error("Failed to get B64 string for inbound", "inb", inb)
			os.Exit(-1)
		}

		logger.Info("For this inbound B64: ", "b64", "tau://"+b64_str)

		var obfs obfuscation.Obfuscator
		var trans transport.Server
		var handsh handshake.HandshakeHandler

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
			os.Exit(-1)
		}

		switch inb.Handsh {
		case "plain":
			handsh = &basic.BasicHandshaker{}
		default:
			logger.Error("Invalid handshake method", "inb", name, "name", inb.Handsh)
			os.Exit(-1)
		}

		go startServer(
			config.BindIP+":"+strconv.Itoa(inb.Port),
			handsh,
			trans,
			obfs,
			logger,
			&wg,
		)
		wg.Add(1)
	}

	wg.Wait()
}

func startServer(addr string, handsh handshake.HandshakeHandler, trans transport.Server, obfs obfuscation.Obfuscator, logger *slog.Logger, wg *sync.WaitGroup) {
	defer wg.Done()

	server := server.NewServer(trans, obfs, handsh, logger)

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

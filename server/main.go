package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/agnostic-t/neutrino-core/core/server"
	"github.com/agnostic-t/neutrino-core/local"
	"github.com/agnostic-t/neutrino-core/obfuscation"
	"github.com/agnostic-t/neutrino-core/transport"
	"github.com/agnostic-t/neutrino-lproxies/socks5"
	"github.com/agnostic-t/neutrino-obfs/xobfs"

	"github.com/agnostic-t/neutrino-transport/basic/tcp"
)

func InitLocalProxy(addr string) local.Proxy {
	return socks5.NewProxy(addr)
}

func InitTCPClient(vpnServerAddr string) transport.Client {
	return tcp.NewClient(vpnServerAddr, time.Second*5)
}

func InitObfs(psk []byte) obfuscation.Obfuscator {
	return &xobfs.Obfuscator{
		Psk: psk,
	}
}

func InitTCPTransport(bindAddr string) transport.Server {
	return tcp.NewServer(bindAddr)
}

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	trans := InitTCPTransport("0.0.0.0:9001")
	obfs := InitObfs([]byte("IkupwyNCJrl<pRSRYrtULW&QA%TXE<"))

	server := server.NewServer(trans, obfs, logger)

	logger.Info("Server is starting at 0.0.0.0:9001")

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	if err := server.Start(ctx); err != nil {
		logger.Error("Failed to start server", "error", err)
	}
}

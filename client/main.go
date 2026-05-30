package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/agnostic-t/neutrino-core/core/client"
	"github.com/agnostic-t/neutrino-core/local"
	"github.com/agnostic-t/neutrino-core/obfuscation"
	"github.com/agnostic-t/neutrino-core/transport"
	"github.com/agnostic-t/neutrino-obfs/xobfs"

	"github.com/agnostic-t/neutrino-lproxies/socks5"
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

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	proxy := InitLocalProxy("127.0.0.1:9002")
	defer proxy.Close()

	trans := InitTCPClient("127.0.0.1:9001")
	obfs := InitObfs([]byte("IkupwyNCJrl<pRSRYrtULW&QA%TXE<"))

	client := client.NewClient(proxy, trans, obfs, logger)

	logger.Info("Starting Neutrino Client on 127.0.0.1:9002")

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	if err := client.Start(ctx); err != nil {
		logger.Error("Failed to start client", "error", err)
	}
}

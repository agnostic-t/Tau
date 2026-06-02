package tun

import (
	_ "github.com/xjasonlyu/tun2socks/v2/dns"
	"github.com/xjasonlyu/tun2socks/v2/engine"
)

// tun2socks --device tun0 --proxy socks5://host:port --interface eth0

func StartTUN2SOCKS(
	tunManager *Manager,
	proxyAddr string,
) {
	key := new(engine.Key)
	key.Device = tunManager.tunIF
	key.Proxy = proxyAddr
	key.Interface = tunManager.primalIF

	engine.Insert(key)

	engine.Start()
}

func StopTUN2SOCKS() {
	engine.Stop()
}

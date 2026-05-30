![[logo]](./assets/logo.png)

# Neutrino project | Basic VPN

Repository implements basic version of vpn, based on neutrino core

Uses all modules from neutrino core (`obfs`, `transport`, `lproxy`)

It has:
- SOCKS5 proxy for connection on client side
- TCP transport as network communication protocol
- XOBFS for obfuscating data

## Usage

Firstly, you have to write a config files for both server:

```json
{
  "bindIP": "0.0.0.0",
  "externalIP": "127.0.0.1",
  "inbounds": {
    "i-123": {
      "port": 9001,
      "obfs": "xobfs",
      "trans": "tcp",
      "handshake": "plain",
      "psk": "123123321321"
    }
  },
  "clients": {}
}
```

and client:

```json
{
  "lproxy": {
    "socks5": "127.0.0.1:9002"
  },
  "selected": "123",
  "servers": {
    "123": {
      "addr": "127.0.0.1:9001",
      "obfs": "xobfs",
      "psk": "123123321321",
      "traffic": "unlimited",
      "handshake": "plain",
      "locked": true
    }
  }
}
```

Than you can start server:

```sh
go run ./server ./config/server.json
```

and than client:

```sh
go run ./client ./config/client.json
```

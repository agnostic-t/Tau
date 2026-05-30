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
  "inbounds": {
    "i-123": {
      "port": 9001,
      "obfs": "xobfs",
      "trans": "tcp",
      "psk": "123123321321"
    }
  },
  "clients": {
    "v-123": {
      "inbound": "i-123",
      "traffic": "unlimited",
      "locked": true
    }
  }
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
      "locked": true
    }
  }
}
```

Than you can start server:

```sh
go run ./server
```

and than client:

```sh
go run ./client
```

Client starts SOCKS5 proxy on localhost:9002 and server starts it's own server on 0.0.0.0:9001

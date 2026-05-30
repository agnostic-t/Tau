![[logo]](./assets/logo.png)

# Neutrino project | Basic VPN

Repository implements basic version of vpn, based on neutrino core

Uses all modules from neutrino core (`obfs`, `transport`, `lproxy`)

It has:
- SOCKS5 proxy for connection on client side
- TCP transport as network communication protocol
- XOBFS for obfuscating data

## Usage

It is very simple. Firstly you start server:

```sh
go run ./server
```

and than client:

```sh
go run ./client
```

Client starts SOCKS5 proxy on localhost:9002 and server starts it's own server on 0.0.0.0:9001

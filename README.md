![[logo]](./assets/tau-logo.png)

English | [Русский](./README_RU.md)

# Tau

A VPN project built on basic implementations of [Neutrino core](https://github.com/agnostic-t/neutrino-core) modules . Tau supports:

- TCP as a data transport over the network
- Null and xOBFS modes for traffic obfuscation
- Plain and xOBFS handshake modes
- Null and Yamux modes for multiplexing
- Generation of ephemeral keys every N seconds without transferring any information between the client and the server (**timeferal** keys)

A description of how the algorithms work is on https://docs.worldfreeteam.org

## Usage

Setting up and deploying Tau is as simple as possible. All together it takes ~5 minutes. On both the client and the server, you first need to install `go` (1.26.3)

### Installation

Installation is reduced to downloading and compiling:

```sh
git clone https://github.com/agnostic-t/tau
cd tau

# For the client
sudo go build client -o /usr/local/bin/taucli ./client/main.go 

# For the server
sudo go build -o /usr/local/bin/tauhost ./server/main.go 
```

Make sure that `/usr/local/bin` is in the PATH (or change the path where to put the binary)


### Setting up

The setup also doesn't differ much between the client and the server. Server's configuration:

```json
{
  "bindIP": "0.0.0.0",
  "externalIP": "1.2.3.4",
  "inbounds": {
    "name": {
      "port": 1234,
      "obfs": {},
      "handshake": {},
      "trans": {},
      "mux": {}
    }
  }
}
```

`bindIP` is an IP for binding server localy and `externalIP` is the IP that can be used to connect to the server from outside.

Detailed settings of the inbounds parameters will be provided below.

Client, it`s more interesting here.:

```json
{
  "lproxy": {
    "socks5": "127.0.0.1:9005"
  },
  "tun": {
    "tun": "tunIF0",
    "main": "mainIF0",
    "gateway": "192.168.1.1",
    "enabled": true
  },
  "filter": {
    "direct": "/path/to/direct/list.txt",
    "block": "/path/to/block/list.txt"
  },
  "selcted": "name",
  "servers": {
    "name": {
      "addr": "1.2.3.4:1234",
      "obfs": {},
      "handshake": {},
      "trans": {},
      "mux": {}
    }
  }
}
```

As you might notice, the client and the server have 4 identical fields (`obfs`, `handshake`, `trans` and `mux`). These fields correspondingly determine which protocols will be used for: obfuscation, handshake, transport, and multiplexation.

Each of the 4 fields is filled in as follows:

```json
"obfs": {
  "type": "...",
  "settings": {
    "arg": "value"
  }
}
```

It is possible to fill in only `type`, without `settings` at all. The following settings are currently available:

1) `obfs`:
    - `type`: `xobfs` or `null`
      - `xobfs` obfuscates traffic, `null` does not. xobfs requires `settings`
    - `settings`: for xobfs it is specified:
      - `psk`: string, obfuscation key

2) `handshake`:
    - `type`: `xobfs` or `plain`
      - `xobfs` changes the number of packets when connecting, has timeferal keys, and obfuscates the target. 
      - `plain` transmits the connection target as plain text
    - `settings`: needed for `xobfs`:
      - `psk`: string, obfuscation key
      - `startJunk`: Boolean value, whether to use junk packets or not
      - `rotateSeconds`: the number of times in how many seconds to change the signature of the handshake
      - `rotateJunkCount`: Boolean value, use timeferal keys to change the number of junk packets or not
      - `minJunkPacks`: number, minimum number of junk packets
      - `maxJunkPacks`: number, maximum number of junk packets

3. `trans`:
    - `type`: currently only `tcp`, no settings required

4. `mux`:
    - `type`: `yamux` and `null` are available. The first uses multiplexing (1 physical transport connection for multiple requests), the second does not use multiplexing, opens multiple connections to the server to transfer information.

### Launch

Starting after building and configuration is trivial:

```sh
taucli /path/to/config.json
tauhost /path/to/config.json
```

> You can wrap the command in `nohup ... &` for the application to work even with the ssh session closed or the terminal closed (at the client)

---

Created using [neutrino-core](https://github.com/agnostic-t/neutrino-core )

![[logo]](./assets/logo.png)

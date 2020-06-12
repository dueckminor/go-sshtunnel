# go-sshtunnel

![build](https://github.com/dueckminor/go-sshtunnel/workflows/build/badge.svg)
[![Go Report Card](https://goreportcard.com/badge/github.com/dueckminor/go-sshtunnel)](https://goreportcard.com/report/github.com/dueckminor/go-sshtunnel)

This is a tiny ssh tunnel implemented in GO. It's main purpose is to establish an SSH connection from a Docker container to a jumpbox and redirect all outgoing TCP traffic over this connection.

To start `sshtunnel` daemon process use:

```bash
sshtunnel start
```

## Proxies

This daemon process can now be used to start various proxies which handle
requests from local clients.

### Proxy-Types

#### TCP-Proxy (Linux only)

```bash
sshtunnel start-proxy tcp
```

#### Socks5-Proxy

```bash
sshtunnel start-proxy socks5
```

#### DNS-Proxy

Listen on a local UDP port and forward DNS requests over TCP to a target address

```bash
sshtunnel start-proxy dns --target=127.0.0.53:53
```

## Rules

Rules are used to select which dialer has to be used for a target address.

## Dialers

Finally the dialers forwards the requests (via SSH) to its destination.

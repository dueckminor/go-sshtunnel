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

The TCP-Proxy listens on a TCP port and allows to forward requests
which have been redirect to this port using the `iptables` feature `--to-ports`.

```bash
sshtunnel start-proxy tcp [<port>]
```

If no port is specified, a random (unused) port will be used.

To do the `iptables` configuration, you have to execute the following command:

```bash
sh <(sshtunnel iptables-script)
```

#### Socks5-Proxy

```bash
sshtunnel start-proxy socks5 [<port>]
```

If no port is specified, a random (unused) port will be used.

#### DNS-Proxy

Listen on a local UDP port and forward DNS requests over TCP to a target address. This allows forwarding of DNS requests via the tunnel.
As the tunnel itself only supports TCP, sshtunnel translates from UDP to TCP.

```bash
sshtunnel start-proxy dns 127.0.0.53:53
```

## Rules

Rules are used to select which dialer has to be used for a target address.

```bash
sshtunnel add-rule <ip-address/network>
```

## Dialers

Finally, the dialers forwards the requests (via SSH) to its destination.

```bash
sshtunnel add-ssh-key <ssh_key_file>
sshtunnel add-dialer [<username>@]<hostname>
```

It's allowed to add multiple ssh dialers:

```bash
sshtunnel add-dialer [<username>@]<hostname>,[<username2>@]<hostname2>
# or
sshtunnel add-dialer [<username>@]<hostname>
sshtunnel add-dialer [<username2>@]<hostname2>
```

It's also possible to use an existing socks5 proxy to establish connections:

```bash
sshtunnel add-dialer socks5://<hostname>:<port>
```

# Release builds

To create a release, you just have to tag a commit with a tag starting with
`v`, push this tag and wait...

```bash
> git tag v1.0-beta4
> git push origin v1.0-beta4
```

Yow will find the released binaries some minutes later on the [Releases](https://github.com/dueckminor/go-sshtunnel/releases) page.

# go-sshtunnel

![build](https://github.com/dueckminor/go-sshtunnel/workflows/build/badge.svg)
[![Go Report Card](https://goreportcard.com/badge/github.com/dueckminor/go-sshtunnel)](https://goreportcard.com/report/github.com/dueckminor/go-sshtunnel)

This is a tiny ssh tunnel implemented in GO. It's main purpose is to establish an SSH connection from a Docker container to a jumpbox and redirect all outgoing TCP traffic over this connection.

Currently only LINUX is supported

Usage:
```
sshtunnel <jumpbox_ip> [networks...] &

Example:
sshtunnel 12.34.56.78 10.0.0.0/8 192.168.1.0/24 &
```

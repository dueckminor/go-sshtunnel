package commands

import (
	"fmt"

	"github.com/dueckminor/go-sshtunnel/control"
)

func init() {
	RegisterCommand("iptables-script", cmdIptablesScript{})
}

type cmdIptablesScript struct{}

func (cmdIptablesScript) Execute(args ...string) error {
	c := control.Client()
	proxies, err := c.ListProxies()
	if err != nil {
		return err
	}
	rules, err := c.ListRules()
	if err != nil {
		return err
	}

	transparentPort := 0
	dnsPort := 0

	for _, proxy := range proxies {
		switch proxy.ProxyType {
		case "transparent":
			transparentPort = proxy.ProxyPort
		case "dns":
			dnsPort = proxy.ProxyPort
		}
	}

	fmt.Print(`#!/usr/bin/env bash
set -e

sudo iptables-save | grep -v sshtunnel | sudo iptables-restore
sudo iptables -t nat -N sshtunnel
sudo iptables -t nat -F sshtunnel
sudo iptables -t nat -I OUTPUT 1 -j sshtunnel
sudo iptables -t nat -I PREROUTING 1 -j sshtunnel
`)

	for _, rule := range rules {
		fmt.Printf("sudo iptables -t nat -A sshtunnel -j REDIRECT --dest %s -p tcp --to-ports %d\n", rule.CIDR, transparentPort)
	}

	if dnsPort > 0 {
		fmt.Printf("sudo iptables -t nat -A sshtunnel -p udp --dport 53 -j REDIRECT --to-ports %d\n", dnsPort)
	}

	return nil
}

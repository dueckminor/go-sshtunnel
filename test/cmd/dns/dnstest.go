package main

import (
	"net"
	"time"

	"github.com/dueckminor/go-sshtunnel/proxy"
)

func main() {
	proxy.NewDNSProxy(&net.Dialer{}, 1053, "192.168.0.1:53")

	time.Sleep(time.Minute * 5)
}

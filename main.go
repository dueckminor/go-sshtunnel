package main

import (
	"fmt"
	"log"
	"net"
	"net/url"
	"os"

	"gopkg.in/alecthomas/kingpin.v2"
)

var (
	showVersion = kingpin.Flag("version", "Show version information of sshtunnel").Bool()
	sshServer   = kingpin.Arg("ssh-server", "URL to ssh server. E.g. user@my.sshserver.com:22").Required().String()
	privateKey  = kingpin.Flag("private-key", "Location of the ssh private key. Default is $HOME/.ssh/id_rsa").
			Default(os.ExpandEnv("$HOME/.ssh/id_rsa")).Short('i').String()
	networks = kingpin.Arg("networks", "List of networks to route via ssh server. Default is 10.0.0.0/8").
			Default("10.0.0.0/8").Strings()
	sshTimeout = kingpin.Flag("timeout", "Set time for ssh connection in second. Default is 10").Default("10").Int()
	dnsServer  = kingpin.Flag("dns", "IP-Address of a DNS server in the tunneled network").String()

	// automatically filled by goreleaser OR manually by go build -ldflags="-X main.version=1.0 ..."
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

var L = log.New(os.Stdout, "sshuttle: ", log.Lshortfile|log.LstdFlags)

func main() {
	kingpin.Parse()
	if *showVersion {
		fmt.Printf("version: %v\ncommit: %v\ndate: %v\n", version, commit, date)
		os.Exit(0)
	}

	tunnel := &SSHTunnel{}
	sshUrl, err := url.Parse("ssh://" + *sshServer)
	if err != nil {
		fmt.Printf("%q is not a valid ssh url: %v", *sshServer, err)
		os.Exit(1)
	}

	if sshUrl.User == nil || sshUrl.User.Username() == "" {
		tunnel.user = os.Getenv("USERNAME")
	} else {
		tunnel.user = sshUrl.User.Username()
	}

	tunnel.port = "22"
	if sshUrl.Port() != "" {
		tunnel.port = sshUrl.Port()
	}
	tunnel.host = sshUrl.Host
	tunnel.privateKey = *privateKey
	tunnel.networks = make([]*net.IPNet, len(*networks))
	tunnel.timeout = *sshTimeout
	if dnsServer != nil {
		tunnel.dns = *dnsServer
	}

	for idx, networkName := range *networks {
		_, network, err := net.ParseCIDR(networkName)
		if err != nil {
			panic(err)
		}
		tunnel.networks[idx] = network
	}

	tunnel.Start()

}

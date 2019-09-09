package main

import (
	"fmt"
	"net"
	"net/url"
	"os"

	"github.com/dueckminor/go-sshtunnel/control"
	"github.com/dueckminor/go-sshtunnel/daemon"
	"github.com/dueckminor/go-sshtunnel/sshtunnel"
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

var sshTunnelDaemon = daemon.Daemon{
	PIDFile: "/tmp/sshtunnel.pid",
}

func run() {
	kingpin.Parse()
	if *showVersion {
		fmt.Printf("version: %v\ncommit: %v\ndate: %v\n", version, commit, date)
		os.Exit(0)
	}

	control.Start()

	tunnel := &sshtunnel.SSHTunnel{}
	sshUrl, err := url.Parse("ssh://" + *sshServer)
	if err != nil {
		fmt.Printf("%q is not a valid ssh url: %v", *sshServer, err)
		os.Exit(1)
	}

	if sshUrl.User == nil || sshUrl.User.Username() == "" {
		tunnel.User = os.Getenv("USERNAME")
	} else {
		tunnel.User = sshUrl.User.Username()
	}

	tunnel.Port = "22"
	if sshUrl.Port() != "" {
		tunnel.Port = sshUrl.Port()
	}
	tunnel.Host = sshUrl.Host
	tunnel.PrivateKey = *privateKey
	tunnel.Networks = make([]*net.IPNet, len(*networks))
	tunnel.Timeout = *sshTimeout
	if dnsServer != nil {
		tunnel.DNS = *dnsServer
	}

	for idx, networkName := range *networks {
		_, network, err := net.ParseCIDR(networkName)
		if err != nil {
			panic(err)
		}
		tunnel.Networks[idx] = network
	}

	tunnel.Start()
}

func main() {
	sshTunnelDaemon.Main(run)
}

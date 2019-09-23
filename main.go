package main

import (
	"fmt"
	"io/ioutil"
	"net"
	"net/url"
	"os"

	"github.com/dueckminor/go-sshtunnel/sshdialer"

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

type controller struct {
	done    chan int
	targets sshtunnel.Targets
	sshDialer *sshdialer.SSHDialer
}

func (c *controller) Start() {
	c.done = make(chan int)
}

func (c *controller) Health() (bool, error) {
	return true, nil
}

func (c *controller) Stop() error {
	c.done <- 0
	return nil
}

func (c *controller) AddSSHKey(encodedKey string, passPhrase string) error {
	return c.sshDialer.AddSSHKey(encodedKey, passPhrase)
}

func (c *controller) AddTarget(cidr string, tunnel string) error {
	return c.targets.AddTarget(cidr, tunnel)
}

func (c *controller) GetConfigScript() (string, error) {
	return "", nil
}

func run() {
	kingpin.Parse()
	if *showVersion {
		fmt.Printf("version: %v\ncommit: %v\ndate: %v\n", version, commit, date)
		os.Exit(0)
	}

	ctrl := &controller{}
	ctrl.Start()

	go control.Start(ctrl)

	tunnel := &sshtunnel.SSHTunnel{}
	sshURL, err := url.Parse("ssh://" + *sshServer)
	if err != nil {
		fmt.Printf("%q is not a valid ssh url: %v", *sshServer, err)
		os.Exit(1)
	}

	if sshURL.User == nil || sshURL.User.Username() == "" {
		tunnel.User = os.Getenv("USERNAME")
	} else {
		tunnel.User = sshURL.User.Username()
	}

	tunnel.Port = "22"
	if sshURL.Port() != "" {
		tunnel.Port = sshURL.Port()
	}
	tunnel.Host = sshURL.Host
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

	ctrl.sshDialer, err = sshdialer.NewSSHDialer(tunnel.Host, tunnel.Port, tunnel.User, tunnel.Timeout)
	if err != nil {
		panic(err)
	}

	go tunnel.Start()

	rc := <-ctrl.done
	os.Exit(rc)
}

func main() {
	cmd := "status"
	if len(os.Args) > 1 {
		cmd = os.Args[1]
	}

	switch cmd {
	case "status":
		h, err := control.Client().Health()
		fmt.Println(h, err)
		return
	case "stop":
		control.Client().Stop()
		return
	case "add-ssh-key":
		encodedKey := ""
		if len(os.Args) > 2 {
			body, err := ioutil.ReadFile(os.Args[2])
			if err != nil {
				panic(err)
			}
			encodedKey = string(body)
			control.Client().AddSSHKey(encodedKey, "")
		}
		return
	}

	sshTunnelDaemon.Main(run)
}

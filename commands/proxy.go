package commands

import (
	"flag"
	"fmt"

	"github.com/dueckminor/go-sshtunnel/control"
)

func init() {
	RegisterCommand("start-proxy", cmdStartProxy{})
	RegisterCommand("list-proxies", cmdListProxies{})
}

type cmdStartProxy struct{}

func (cmdStartProxy) Execute(args ...string) error {
	parameters := ""
	if len(args) > 1 {
		parameters = args[1]
	}

	flag.Parse()

	_, err := control.Client().StartProxy(args[0], parameters)
	return err
}

type cmdListProxies struct{}

func (cmdListProxies) Execute(args ...string) error {
	proxies, err := control.Client().ListProxies()
	if err != nil {
		fmt.Println(err)
		return err
	}

	if len(proxies) == 0 {
		fmt.Println("no proxies started")
		return nil
	}

	fmt.Println("the following proxies are running:")
	for _, proxy := range proxies {
		fmt.Printf("- %s (port: %d)\n", proxy.ProxyType, proxy.ProxyPort)
	}

	return err
}

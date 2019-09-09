package commands

import (
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/dueckminor/go-sshtunnel/control"
)

func init() {
	RegisterCommand("add-dialer", cmdAddDialer{})
	RegisterCommand("list-dialers", cmdListDialers{})
}

////////////////////////////////////////////////////////////////////////////////

type cmdAddDialer struct{}

func (cmdAddDialer) Execute(args ...string) error {
	sshServer := args[0]
	if strings.HasPrefix(sshServer, "socks5://") {
		return control.Client().AddDialer(sshServer)
	}

	sshURL, err := url.Parse("ssh://" + sshServer)
	if err != nil {
		fmt.Printf("%s is not a valid ssh url: %v", sshServer, err)
		os.Exit(1)
	}

	if sshURL.User == nil || sshURL.User.Username() == "" {
		username := os.Getenv("USERNAME")
		if username == "" {
			username = os.Getenv("LOGNAME")
		}
		sshURL.User = url.User(username)
	}

	uri := sshURL.String()

	fmt.Println("Adding dialer:", uri)

	return control.Client().AddDialer(uri)
}

////////////////////////////////////////////////////////////////////////////////

type cmdListDialers struct{}

func (cmdListDialers) Execute(args ...string) error {
	return nil
}

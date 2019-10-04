package commands

import (
	"fmt"
	"net/url"
	"os"

	"github.com/dueckminor/go-sshtunnel/control"
)

func init() {
	RegisterCommand("add-dialer", cmdAddDialer{})
}

type cmdAddDialer struct{}

func (cmdAddDialer) Execute(args ...string) error {
	sshServer := args[0]

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

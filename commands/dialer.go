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

	var uris []string

	for _, sshsshServerPart := range strings.Split(sshServer, ",") {
		if !strings.Contains(sshsshServerPart, "://") {
			sshsshServerPart = "ssh://" + sshsshServerPart
		}
		sshURL, err := url.Parse(sshsshServerPart)
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

		uris = append(uris, sshURL.String())
	}

	uri := strings.Join(uris, ",")
	fmt.Println("Adding dialer:", uri)

	return control.Client().AddDialer(uri)
}

////////////////////////////////////////////////////////////////////////////////

type cmdListDialers struct{}

func (cmdListDialers) Execute(args ...string) error {
	dialers, err := control.Client().ListDialers()
	if err != nil {
		return err
	}
	if len(dialers) == 0 {
		fmt.Println("dialers: []")
	}
	fmt.Println("dialers:")
	for _, dialer := range dialers {
		fmt.Printf("  - name: %s\n", dialer.Name)
		fmt.Printf("    type: %s\n", dialer.Type)
		fmt.Printf("    destination: %s\n", dialer.Destination)
	}
	return nil
}

package commands

import (
	"fmt"
	"os"

	"github.com/dueckminor/go-sshtunnel/control"
)

func init() {
	RegisterCommand("status", cmdStatus{})
}

type cmdStatus struct{}

func (cmdStatus) Execute(args ...string) error {
	status, err := control.Client().Status()
	fmt.Println(status, err)
	if !status.Healthy {
		os.Exit(1)
	}
	return err
}

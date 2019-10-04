package commands

import (
	"fmt"

	"github.com/dueckminor/go-sshtunnel/control"
)

func init() {
	RegisterCommand("status", cmdStatus{})
}

type cmdStatus struct{}

func (cmdStatus) Execute(args ...string) error {
	status, err := control.Client().Status()
	fmt.Println(status, err)
	return err
}

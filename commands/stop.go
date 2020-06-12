package commands

import "github.com/dueckminor/go-sshtunnel/control"

func init() {
	RegisterCommand("stop", cmdStop{})
}

type cmdStop struct{}

func (cmdStop) Execute(args ...string) error {
	control.Client().Stop()
	return nil
}

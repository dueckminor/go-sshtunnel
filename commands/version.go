package commands

import (
	"fmt"
)

var (
	version = "dev-build"
)

func init() {
	RegisterCommand("version", cmdVersion{})
}

type cmdVersion struct{}

func (cmdVersion) Execute(args ...string) error {
	fmt.Println("sshtunnel version:", version)
	return nil
}

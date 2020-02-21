package main

import (
	"os"

	"github.com/dueckminor/go-sshtunnel/server"

	"github.com/dueckminor/go-sshtunnel/commands"
)

func main() {
	cmd := "status"
	parameters := []string{}
	if len(os.Args) > 1 {
		cmd = os.Args[1]
		parameters = os.Args[2:]
	}

	switch cmd {
	case "daemon":
		server.Run(parameters)
		return
	case "start":
		server.Start(parameters)
		return
	case "stop":
		server.Stop()
		return
	}

	commands.ExecuteCommand(cmd, parameters...)
}

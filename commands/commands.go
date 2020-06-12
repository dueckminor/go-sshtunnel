package commands

import (
	"fmt"
	"os"
	"sort"
)

// Command is the interface for a Command
type Command interface {
	Execute(args ...string) error
}

var (
	commandMap = make(map[string]Command)
)

// RegisterCommand makes a command available for the CLI
func RegisterCommand(commandName string, command Command) {
	commandMap[commandName] = command
}

func usage(commandName string) {
	if command, ok := commandMap[commandName]; ok {
		fmt.Println(command)
	} else {
		fmt.Printf("Usage: %s [subcommand] [arguments...]\n", os.Args[0])
		fmt.Println("where subcommand is one of:")

		var commandNames []string
		for k := range commandMap {
			commandNames = append(commandNames, k)
		}
		sort.Strings(commandNames)

		for _, commandName := range commandNames {
			fmt.Println("-", commandName)
		}
	}
	os.Exit(1)
}

// ExecuteCommand executes a command by name
func ExecuteCommand(commandName string, args ...string) error {
	if command, ok := commandMap[commandName]; ok {
		return command.Execute(args...)
	}
	usage(commandName)
	return nil
}

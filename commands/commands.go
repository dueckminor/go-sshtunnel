package commands

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

// ExecuteCommand executes a command by name
func ExecuteCommand(commandName string, args ...string) error {
	if command, ok := commandMap[commandName]; ok {
		return command.Execute(args...)
	}
	return nil
}

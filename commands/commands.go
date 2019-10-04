package commands

type Command interface {
	Execute(args ...string) error
}

var (
	commandMap = make(map[string]Command)
)

func RegisterCommand(commandName string, command Command) {
	commandMap[commandName] = command
}

func ExecuteCommand(commandName string, args ...string) error {
	if command, ok := commandMap[commandName]; ok {
		return command.Execute(args...)
	}
	return nil
}

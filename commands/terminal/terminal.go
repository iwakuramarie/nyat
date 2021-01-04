package terminal

import (
	"gitea.com/iwakuramarie/nyat/commands"
)

var (
	TerminalCommands *commands.Commands
)

func register(cmd commands.Command) {
	if TerminalCommands == nil {
		TerminalCommands = commands.NewCommands()
	}
	TerminalCommands.Register(cmd)
}

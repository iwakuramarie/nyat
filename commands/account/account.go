package account

import (
	"gitea.com/iwakuramarie/nyat/commands"
)

var (
	AccountCommands *commands.Commands
)

func register(cmd commands.Command) {
	if AccountCommands == nil {
		AccountCommands = commands.NewCommands()
	}
	AccountCommands.Register(cmd)
}

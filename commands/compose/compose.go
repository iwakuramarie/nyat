package compose

import (
	"gitea.com/iwakuramarie/nyat/commands"
)

var (
	ComposeCommands *commands.Commands
)

func register(cmd commands.Command) {
	if ComposeCommands == nil {
		ComposeCommands = commands.NewCommands()
	}
	ComposeCommands.Register(cmd)
}

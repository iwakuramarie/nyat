package msg

import (
	"gitea.com/iwakuramarie/nyat/commands"
)

var (
	MessageCommands *commands.Commands
)

func register(cmd commands.Command) {
	if MessageCommands == nil {
		MessageCommands = commands.NewCommands()
	}
	MessageCommands.Register(cmd)
}

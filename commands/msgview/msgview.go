package msgview

import (
	"gitea.com/iwakuramarie/nyat/commands"
)

var (
	MessageViewCommands *commands.Commands
)

func register(cmd commands.Command) {
	if MessageViewCommands == nil {
		MessageViewCommands = commands.NewCommands()
	}
	MessageViewCommands.Register(cmd)
}

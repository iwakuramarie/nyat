package commands

import (
	"fmt"

	"gitea.com/iwakuramarie/nyat/widgets"
)

type Prompt struct{}

func init() {
	register(Prompt{})
}

func (Prompt) Aliases() []string {
	return []string{"prompt"}
}

func (Prompt) Complete(nyat *widgets.Nyat, args []string) []string {
	return nil // TODO: add completions
}

func (Prompt) Execute(nyat *widgets.Nyat, args []string) error {
	if len(args) < 3 {
		return fmt.Errorf("Usage: %s <prompt> <cmd>", args[0])
	}

	prompt := args[1]
	cmd := args[2:]
	nyat.RegisterPrompt(prompt, cmd)
	return nil
}

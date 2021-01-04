package commands

import (
	"errors"

	"gitea.com/iwakuramarie/nyat/widgets"
)

type Help struct{}

func init() {
	register(Help{})
}

func (Help) Aliases() []string {
	return []string{"help"}
}

func (Help) Complete(nyat *widgets.Nyat, args []string) []string {
	return nil
}

func (Help) Execute(nyat *widgets.Nyat, args []string) error {
	page := "nyat"
	if len(args) == 2 {
		page = "nyat-" + args[1]
	} else if len(args) > 2 {
		return errors.New("Usage: help [topic]")
	}
	return TermCore(nyat, []string{"term", "man", page})
}

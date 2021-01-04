package commands

import (
	"errors"

	"gitea.com/iwakuramarie/nyat/widgets"
)

type Quit struct{}

func init() {
	register(Quit{})
}

func (Quit) Aliases() []string {
	return []string{"quit", "exit"}
}

func (Quit) Complete(nyat *widgets.Nyat, args []string) []string {
	return nil
}

type ErrorExit int

func (err ErrorExit) Error() string {
	return "exit"
}

func (Quit) Execute(nyat *widgets.Nyat, args []string) error {
	if len(args) != 1 {
		return errors.New("Usage: quit")
	}
	return ErrorExit(1)
}

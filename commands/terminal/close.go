package terminal

import (
	"errors"

	"gitea.com/iwakuramarie/nyat/widgets"
)

type Close struct{}

func init() {
	register(Close{})
}

func (Close) Aliases() []string {
	return []string{"close"}
}

func (Close) Complete(nyat *widgets.Nyat, args []string) []string {
	return nil
}

func (Close) Execute(nyat *widgets.Nyat, args []string) error {
	if len(args) != 1 {
		return errors.New("Usage: close")
	}
	term, _ := nyat.SelectedTab().(*widgets.Terminal)
	term.Close(nil)
	return nil
}

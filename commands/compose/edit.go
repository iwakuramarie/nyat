package compose

import (
	"errors"

	"gitea.com/iwakuramarie/nyat/widgets"
)

type Edit struct{}

func init() {
	register(Edit{})
}

func (Edit) Aliases() []string {
	return []string{"edit"}
}

func (Edit) Complete(nyat *widgets.Nyat, args []string) []string {
	return nil
}

func (Edit) Execute(nyat *widgets.Nyat, args []string) error {
	if len(args) != 1 {
		return errors.New("Usage: edit")
	}
	composer, _ := nyat.SelectedTab().(*widgets.Composer)
	composer.ShowTerminal()
	composer.FocusTerminal()
	return nil
}

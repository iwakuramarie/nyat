package compose

import (
	"errors"

	"gitea.com/iwakuramarie/nyat/widgets"
)

type Abort struct{}

func init() {
	register(Abort{})
}

func (Abort) Aliases() []string {
	return []string{"abort"}
}

func (Abort) Complete(nyat *widgets.Nyat, args []string) []string {
	return nil
}

func (Abort) Execute(nyat *widgets.Nyat, args []string) error {
	if len(args) != 1 {
		return errors.New("Usage: abort")
	}
	composer, _ := nyat.SelectedTab().(*widgets.Composer)

	nyat.RemoveTab(composer)
	composer.Close()

	return nil
}

package msgview

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
	mv, _ := nyat.SelectedTab().(*widgets.MessageViewer)
	mv.Close()
	nyat.RemoveTab(mv)
	return nil
}

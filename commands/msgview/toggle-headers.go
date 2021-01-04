package msgview

import (
	"fmt"

	"gitea.com/iwakuramarie/nyat/widgets"
)

type ToggleHeaders struct{}

func init() {
	register(ToggleHeaders{})
}

func (ToggleHeaders) Aliases() []string {
	return []string{"toggle-headers"}
}

func (ToggleHeaders) Complete(nyat *widgets.Nyat, args []string) []string {
	return nil
}

func (ToggleHeaders) Execute(nyat *widgets.Nyat, args []string) error {
	if len(args) > 1 {
		return toggleHeadersUsage(args[0])
	}
	mv, _ := nyat.SelectedTab().(*widgets.MessageViewer)
	mv.ToggleHeaders()
	return nil
}

func toggleHeadersUsage(cmd string) error {
	return fmt.Errorf("Usage: %s", cmd)
}

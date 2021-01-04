package commands

import (
	"fmt"

	"gitea.com/iwakuramarie/nyat/widgets"
)

type PinTab struct{}

func init() {
	register(PinTab{})
}

func (PinTab) Aliases() []string {
	return []string{"pin-tab", "unpin-tab"}
}

func (PinTab) Complete(nyat *widgets.Nyat, args []string) []string {
	return nil
}

func (PinTab) Execute(nyat *widgets.Nyat, args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("Usage: %s", args[0])
	}

	switch args[0] {
	case "pin-tab":
		nyat.PinTab()
	case "unpin-tab":
		nyat.UnpinTab()
	}

	return nil
}

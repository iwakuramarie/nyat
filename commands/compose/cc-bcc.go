package compose

import (
	"strings"

	"gitea.com/iwakuramarie/nyat/widgets"
)

type CC struct{}

func init() {
	register(CC{})
}

func (CC) Aliases() []string {
	return []string{"cc", "bcc"}
}

func (CC) Complete(nyat *widgets.Nyat, args []string) []string {
	return nil
}

func (CC) Execute(nyat *widgets.Nyat, args []string) error {
	var addrs string
	if len(args) > 1 {
		addrs = strings.Join(args[1:], " ")
	}
	composer, _ := nyat.SelectedTab().(*widgets.Composer)

	switch args[0] {
	case "cc":
		composer.AddEditor("Cc", addrs, true)
	case "bcc":
		composer.AddEditor("Bcc", addrs, true)
	}

	return nil
}

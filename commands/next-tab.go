package commands

import (
	"fmt"
	"strconv"

	"gitea.com/iwakuramarie/nyat/widgets"
)

type NextPrevTab struct{}

func init() {
	register(NextPrevTab{})
}

func (NextPrevTab) Aliases() []string {
	return []string{"next-tab", "prev-tab"}
}

func (NextPrevTab) Complete(nyat *widgets.Nyat, args []string) []string {
	return nil
}

func (NextPrevTab) Execute(nyat *widgets.Nyat, args []string) error {
	if len(args) > 2 {
		return nextPrevTabUsage(args[0])
	}
	var (
		n   int = 1
		err error
	)
	if len(args) > 1 {
		n, err = strconv.Atoi(args[1])
		if err != nil {
			return nextPrevTabUsage(args[0])
		}
	}
	for ; n > 0; n-- {
		if args[0] == "prev-tab" {
			nyat.PrevTab()
		} else {
			nyat.NextTab()
		}
	}
	return nil
}

func nextPrevTabUsage(cmd string) error {
	return fmt.Errorf("Usage: %s [n]", cmd)
}

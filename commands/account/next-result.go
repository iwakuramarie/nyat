package account

import (
	"errors"
	"fmt"

	"gitea.com/iwakuramarie/nyat/widgets"
)

type NextPrevResult struct{}

func init() {
	register(NextPrevResult{})
}

func (NextPrevResult) Aliases() []string {
	return []string{"next-result", "prev-result"}
}

func (NextPrevResult) Complete(nyat *widgets.Nyat, args []string) []string {
	return nil
}

func (NextPrevResult) Execute(nyat *widgets.Nyat, args []string) error {
	if len(args) > 1 {
		return nextPrevResultUsage(args[0])
	}
	acct := nyat.SelectedAccount()
	if acct == nil {
		return errors.New("No account selected")
	}
	if args[0] == "prev-result" {
		store := acct.Store()
		if store != nil {
			store.PrevResult()
		}
		acct.Messages().Invalidate()
	} else {
		store := acct.Store()
		if store != nil {
			store.NextResult()
		}
		acct.Messages().Invalidate()
	}
	return nil
}

func nextPrevResultUsage(cmd string) error {
	return fmt.Errorf("Usage: %s [<n>[%%]]", cmd)
}

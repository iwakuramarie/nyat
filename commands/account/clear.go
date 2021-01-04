package account

import (
	"errors"
	"gitea.com/iwakuramarie/nyat/widgets"
	"time"
)

type Clear struct{}

func init() {
	register(Clear{})
}

func (Clear) Aliases() []string {
	return []string{"clear"}
}

func (Clear) Complete(nyat *widgets.Nyat, args []string) []string {
	return nil
}

func (Clear) Execute(nyat *widgets.Nyat, args []string) error {
	acct := nyat.SelectedAccount()
	if acct == nil {
		return errors.New("No account selected")
	}
	store := acct.Store()
	if store == nil {
		return errors.New("Cannot perform action. Messages still loading")
	}
	store.ApplyClear()
	nyat.PushStatus("Clear complete.", 10*time.Second)
	return nil
}

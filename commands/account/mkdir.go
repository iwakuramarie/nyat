package account

import (
	"errors"
	"strings"
	"time"

	"gitea.com/iwakuramarie/nyat/widgets"
	"gitea.com/iwakuramarie/nyat/worker/types"
)

type MakeDir struct{}

func init() {
	register(MakeDir{})
}

func (MakeDir) Aliases() []string {
	return []string{"mkdir"}
}

func (MakeDir) Complete(nyat *widgets.Nyat, args []string) []string {
	return nil
}

func (MakeDir) Execute(nyat *widgets.Nyat, args []string) error {
	if len(args) == 0 {
		return errors.New("Usage: :mkdir <name>")
	}
	acct := nyat.SelectedAccount()
	if acct == nil {
		return errors.New("No account selected")
	}
	name := strings.Join(args[1:], " ")
	acct.Worker().PostAction(&types.CreateDirectory{
		Directory: name,
	}, func(msg types.WorkerMessage) {
		switch msg := msg.(type) {
		case *types.Done:
			nyat.PushStatus("Directory created.", 10*time.Second)
			acct.Directories().Select(name)
		case *types.Error:
			nyat.PushError(" " + msg.Error.Error())
		}
	})
	return nil
}

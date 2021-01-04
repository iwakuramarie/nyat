package msgview

import (
	"gitea.com/iwakuramarie/nyat/commands/account"
	"gitea.com/iwakuramarie/nyat/lib"
	"gitea.com/iwakuramarie/nyat/widgets"
)

type NextPrevMsg struct{}

func init() {
	register(NextPrevMsg{})
}

func (NextPrevMsg) Aliases() []string {
	return []string{"next", "next-message", "prev", "prev-message"}
}

func (NextPrevMsg) Complete(nyat *widgets.Nyat, args []string) []string {
	return nil
}

func (NextPrevMsg) Execute(nyat *widgets.Nyat, args []string) error {
	n, pct, err := account.ParseNextPrevMessage(args)
	if err != nil {
		return err
	}
	mv, _ := nyat.SelectedTab().(*widgets.MessageViewer)
	acct := mv.SelectedAccount()
	store := mv.Store()
	err = account.ExecuteNextPrevMessage(args, acct, pct, n)
	if err != nil {
		return err
	}
	nextMsg := store.Selected()
	if nextMsg == nil {
		nyat.RemoveTab(mv)
		return nil
	}
	lib.NewMessageStoreView(nextMsg, store, nyat.DecryptKeys,
		func(view lib.MessageView, err error) {
			if err != nil {
				nyat.PushError(err.Error())
				return
			}
			nextMv := widgets.NewMessageViewer(acct, nyat.Config(), view)
			nyat.ReplaceTab(mv, nextMv, nextMsg.Envelope.Subject)
		})
	return nil
}

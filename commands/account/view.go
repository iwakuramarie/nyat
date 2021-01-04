package account

import (
	"errors"

	"gitea.com/iwakuramarie/nyat/lib"
	"gitea.com/iwakuramarie/nyat/widgets"
)

type ViewMessage struct{}

func init() {
	register(ViewMessage{})
}

func (ViewMessage) Aliases() []string {
	return []string{"view-message", "view"}
}

func (ViewMessage) Complete(nyat *widgets.Nyat, args []string) []string {
	return nil
}

func (ViewMessage) Execute(nyat *widgets.Nyat, args []string) error {
	if len(args) != 1 {
		return errors.New("Usage: view-message")
	}
	acct := nyat.SelectedAccount()
	if acct.Messages().Empty() {
		return nil
	}
	store := acct.Messages().Store()
	msg := acct.Messages().Selected()
	if msg == nil {
		return nil
	}
	_, deleted := store.Deleted[msg.Uid]
	if deleted {
		return nil
	}
	if msg.Error != nil {
		nyat.PushError(msg.Error.Error())
		return nil
	}
	lib.NewMessageStoreView(msg, store, nyat.DecryptKeys,
		func(view lib.MessageView, err error) {
			if err != nil {
				nyat.PushError(err.Error())
				return
			}
			viewer := widgets.NewMessageViewer(acct, nyat.Config(), view)
			nyat.NewTab(viewer, msg.Envelope.Subject)
		})
	return nil
}

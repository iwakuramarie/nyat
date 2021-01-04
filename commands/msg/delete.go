package msg

import (
	"errors"
	"time"

	"gitea.com/iwakuramarie/nyat/lib"
	"gitea.com/iwakuramarie/nyat/models"
	"gitea.com/iwakuramarie/nyat/widgets"
	"gitea.com/iwakuramarie/nyat/worker/types"
)

type Delete struct{}

func init() {
	register(Delete{})
}

func (Delete) Aliases() []string {
	return []string{"delete", "delete-message"}
}

func (Delete) Complete(nyat *widgets.Nyat, args []string) []string {
	return nil
}

func (Delete) Execute(nyat *widgets.Nyat, args []string) error {
	if len(args) != 1 {
		return errors.New("Usage: :delete")
	}

	h := newHelper(nyat)
	store, err := h.store()
	if err != nil {
		return err
	}
	uids, err := h.markedOrSelectedUids()
	if err != nil {
		return err
	}
	acct, err := h.account()
	if err != nil {
		return err
	}
	store.Delete(uids, func(msg types.WorkerMessage) {
		switch msg := msg.(type) {
		case *types.Done:
			nyat.PushStatus("Messages deleted.", 10*time.Second)
		case *types.Error:
			nyat.PushError(" " + msg.Error.Error())
		case *types.Unsupported:
			// notmuch doesn't support it, we want the user to know
			nyat.PushError(" error, unsupported for this worker")
		}
	})

	//caution, can be nil
	next := findNextNonDeleted(uids, store)

	mv, isMsgView := h.msgProvider.(*widgets.MessageViewer)
	if isMsgView {
		if !nyat.Config().Ui.NextMessageOnDelete {
			nyat.RemoveTab(h.msgProvider)
		} else {
			// no more messages in the list
			if next == nil {
				nyat.RemoveTab(h.msgProvider)
				acct.Messages().Invalidate()
				return nil
			}
			lib.NewMessageStoreView(next, store, nyat.DecryptKeys,
				func(view lib.MessageView, err error) {
					if err != nil {
						nyat.PushError(err.Error())
						return
					}
					nextMv := widgets.NewMessageViewer(acct, nyat.Config(), view)
					nyat.ReplaceTab(mv, nextMv, next.Envelope.Subject)
				})
		}
	}
	acct.Messages().Invalidate()
	return nil
}

func findNextNonDeleted(deleted []uint32, store *lib.MessageStore) *models.MessageInfo {
	selected := store.Selected()
	if !contains(deleted, selected.Uid) {
		return selected
	}
	for {
		store.Next()
		next := store.Selected()
		if next == selected || next == nil {
			// the last message is in the deleted state or doesn't exist
			return nil
		}
		return next
	}
	return nil // Never reached
}

func contains(uids []uint32, uid uint32) bool {
	for _, item := range uids {
		if item == uid {
			return true
		}
	}
	return false
}

package imap

import (
	"github.com/emersion/go-imap"

	"gitea.com/iwakuramarie/nyat/models"
	"gitea.com/iwakuramarie/nyat/worker/types"
)

func (imapw *IMAPWorker) handleListDirectories(msg *types.ListDirectories) {
	mailboxes := make(chan *imap.MailboxInfo)
	imapw.worker.Logger.Println("Listing mailboxes")
	done := make(chan interface{})

	go func() {
		for mbox := range mailboxes {
			if !canOpen(mbox) {
				// no need to pass this to handlers if it can't be opened
				continue
			}
			imapw.worker.PostMessage(&types.Directory{
				Message: types.RespondTo(msg),
				Dir: &models.Directory{
					Name:       mbox.Name,
					Attributes: mbox.Attributes,
				},
			}, nil)
		}
		done <- nil
	}()

	if err := imapw.client.List("", "*", mailboxes); err != nil {
		<-done
		imapw.worker.PostMessage(&types.Error{
			Message: types.RespondTo(msg),
			Error:   err,
		}, nil)
	} else {
		<-done
		imapw.worker.PostMessage(
			&types.Done{types.RespondTo(msg)}, nil)
	}
}

func canOpen(mbox *imap.MailboxInfo) bool {
	for _, attr := range mbox.Attributes {
		if attr == imap.NoSelectAttr {
			return false
		}
	}
	return true
}

func (imapw *IMAPWorker) handleSearchDirectory(msg *types.SearchDirectory) {
	emitError := func(err error) {
		imapw.worker.PostMessage(&types.Error{
			Message: types.RespondTo(msg),
			Error:   err,
		}, nil)
	}

	imapw.worker.Logger.Println("Executing search")
	criteria, err := parseSearch(msg.Argv)
	if err != nil {
		emitError(err)
		return
	}

	uids, err := imapw.client.UidSearch(criteria)
	if err != nil {
		emitError(err)
		return
	}

	imapw.worker.PostMessage(&types.SearchResults{
		Message: types.RespondTo(msg),
		Uids:    uids,
	}, nil)

}

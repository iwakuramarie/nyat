package imap

import (
	"io"

	"gitea.com/iwakuramarie/nyat/worker/types"
)

func (imapw *IMAPWorker) handleCopyMessages(msg *types.CopyMessages) {
	uids := toSeqSet(msg.Uids)
	if err := imapw.client.UidCopy(uids, msg.Destination); err != nil {
		imapw.worker.PostMessage(&types.Error{
			Message: types.RespondTo(msg),
			Error:   err,
		}, nil)
	} else {
		imapw.worker.PostMessage(&types.Done{types.RespondTo(msg)}, nil)
	}
}

type appendLiteral struct {
	io.Reader
	Length int
}

func (m appendLiteral) Len() int {
	return m.Length
}

func (imapw *IMAPWorker) handleAppendMessage(msg *types.AppendMessage) {
	if err := imapw.client.Append(msg.Destination, translateFlags(msg.Flags), msg.Date,
		&appendLiteral{
			Reader: msg.Reader,
			Length: msg.Length,
		}); err != nil {

		imapw.worker.PostMessage(&types.Error{
			Message: types.RespondTo(msg),
			Error:   err,
		}, nil)
	} else {
		imapw.worker.PostMessage(&types.Done{types.RespondTo(msg)}, nil)
	}
}

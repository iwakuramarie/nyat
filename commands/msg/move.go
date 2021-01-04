package msg

import (
	"errors"
	"strings"
	"time"

	"git.sr.ht/~sircmpwn/getopt"

	"gitea.com/iwakuramarie/nyat/commands"
	"gitea.com/iwakuramarie/nyat/widgets"
	"gitea.com/iwakuramarie/nyat/worker/types"
)

type Move struct{}

func init() {
	register(Move{})
}

func (Move) Aliases() []string {
	return []string{"mv", "move"}
}

func (Move) Complete(nyat *widgets.Nyat, args []string) []string {
	return commands.GetFolders(nyat, args)
}

func (Move) Execute(nyat *widgets.Nyat, args []string) error {
	if len(args) == 1 {
		return errors.New("Usage: mv [-p] <folder>")
	}
	opts, optind, err := getopt.Getopts(args, "p")
	if err != nil {
		return err
	}
	var (
		createParents bool
	)
	for _, opt := range opts {
		switch opt.Option {
		case 'p':
			createParents = true
		}
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
	_, isMsgView := h.msgProvider.(*widgets.MessageViewer)
	if isMsgView {
		nyat.RemoveTab(h.msgProvider)
	}
	store.Next()
	acct.Messages().Invalidate()
	joinedArgs := strings.Join(args[optind:], " ")
	store.Move(uids, joinedArgs, createParents, func(
		msg types.WorkerMessage) {

		switch msg := msg.(type) {
		case *types.Done:
			nyat.PushStatus("Message moved to "+joinedArgs, 10*time.Second)
		case *types.Error:
			nyat.PushError(" " + msg.Error.Error())
		}
	})
	return nil
}

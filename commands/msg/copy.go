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

type Copy struct{}

func init() {
	register(Copy{})
}

func (Copy) Aliases() []string {
	return []string{"cp", "copy"}
}

func (Copy) Complete(nyat *widgets.Nyat, args []string) []string {
	return commands.GetFolders(nyat, args)
}

func (Copy) Execute(nyat *widgets.Nyat, args []string) error {
	if len(args) == 1 {
		return errors.New("Usage: cp [-p] <folder>")
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
	uids, err := h.markedOrSelectedUids()
	if err != nil {
		return err
	}
	store, err := h.store()
	if err != nil {
		return err
	}
	store.Copy(uids, strings.Join(args[optind:], " "),
		createParents, func(
			msg types.WorkerMessage) {

			switch msg := msg.(type) {
			case *types.Done:
				nyat.PushStatus("Messages copied.", 10*time.Second)
			case *types.Error:
				nyat.PushError(" " + msg.Error.Error())
			}
		})
	return nil
}

package account

import (
	"errors"
	"time"

	"git.sr.ht/~sircmpwn/getopt"

	"gitea.com/iwakuramarie/nyat/widgets"
	"gitea.com/iwakuramarie/nyat/worker/types"
)

type RemoveDir struct{}

func init() {
	register(RemoveDir{})
}

func (RemoveDir) Aliases() []string {
	return []string{"rmdir"}
}

func (RemoveDir) Complete(nyat *widgets.Nyat, args []string) []string {
	return nil
}

func (RemoveDir) Execute(nyat *widgets.Nyat, args []string) error {
	acct := nyat.SelectedAccount()
	if acct == nil {
		return errors.New("No account selected")
	}

	force := false

	opts, optind, err := getopt.Getopts(args, "f")
	if err != nil {
		return err
	}
	for _, opt := range opts {
		switch opt.Option {
		case 'f':
			force = true
		}
	}

	if len(args) != optind {
		return errors.New("Usage: rmdir [-f]")
	}

	// Check for any messages in the directory.
	if !acct.Messages().Empty() && !force {
		return errors.New("Refusing to remove non-empty directory; use -f")
	}

	curDir := acct.SelectedDirectory()
	var newDir string
	dirFound := false

	if oldDir, ok := history[acct.Name()]; ok {
		if oldDir != curDir {
			newDir = oldDir
			dirFound = true
		}
	}

	if !dirFound {
		for _, dir := range acct.Directories().List() {
			if dir != curDir {
				newDir = dir
				dirFound = true
				break
			}
		}
	}

	if !dirFound {
		return errors.New("No directory to move to afterwards!")
	}

	acct.Directories().Select(newDir)

	acct.Worker().PostAction(&types.RemoveDirectory{
		Directory: curDir,
	}, func(msg types.WorkerMessage) {
		switch msg := msg.(type) {
		case *types.Done:
			nyat.PushStatus("Directory removed.", 10*time.Second)
		case *types.Error:
			nyat.PushError(" " + msg.Error.Error())
		case *types.Unsupported:
			nyat.PushError(":rmdir is not supported by the backend.")
		}
	})

	return nil
}

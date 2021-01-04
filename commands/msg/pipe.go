package msg

import (
	"errors"
	"fmt"
	"io"
	"os/exec"
	"time"

	"gitea.com/iwakuramarie/nyat/commands"
	"gitea.com/iwakuramarie/nyat/widgets"
	"gitea.com/iwakuramarie/nyat/worker/types"

	"git.sr.ht/~sircmpwn/getopt"
)

type Pipe struct{}

func init() {
	register(Pipe{})
}

func (Pipe) Aliases() []string {
	return []string{"pipe"}
}

func (Pipe) Complete(nyat *widgets.Nyat, args []string) []string {
	return nil
}

func (Pipe) Execute(nyat *widgets.Nyat, args []string) error {
	var (
		background bool
		pipeFull   bool
		pipePart   bool
	)
	// TODO: let user specify part by index or preferred mimetype
	opts, optind, err := getopt.Getopts(args, "bmp")
	if err != nil {
		return err
	}
	for _, opt := range opts {
		switch opt.Option {
		case 'b':
			background = true
		case 'm':
			if pipePart {
				return errors.New("-m and -p are mutually exclusive")
			}
			pipeFull = true
		case 'p':
			if pipeFull {
				return errors.New("-m and -p are mutually exclusive")
			}
			pipePart = true
		}
	}
	cmd := args[optind:]
	if len(cmd) == 0 {
		return errors.New("Usage: pipe [-mp] <cmd> [args...]")
	}

	provider := nyat.SelectedTab().(widgets.ProvidesMessage)
	if !pipeFull && !pipePart {
		if _, ok := provider.(*widgets.MessageViewer); ok {
			pipePart = true
		} else if _, ok := provider.(*widgets.AccountView); ok {
			pipeFull = true
		} else {
			return errors.New(
				"Neither -m nor -p specified and cannot infer default")
		}
	}

	doTerm := func(reader io.Reader, name string) {
		term, err := commands.QuickTerm(nyat, cmd, reader)
		if err != nil {
			nyat.PushError(" " + err.Error())
			return
		}
		nyat.NewTab(term, name)
	}

	doExec := func(reader io.Reader) {
		ecmd := exec.Command(cmd[0], cmd[1:]...)
		pipe, err := ecmd.StdinPipe()
		if err != nil {
			return
		}
		go func() {
			defer pipe.Close()
			io.Copy(pipe, reader)
		}()
		err = ecmd.Run()
		if err != nil {
			nyat.PushError(" " + err.Error())
		} else {
			if ecmd.ProcessState.ExitCode() != 0 {
				nyat.PushError(fmt.Sprintf(
					"%s: completed with status %d", cmd[0],
					ecmd.ProcessState.ExitCode()))
			} else {
				nyat.PushStatus(fmt.Sprintf(
					"%s: completed with status %d", cmd[0],
					ecmd.ProcessState.ExitCode()), 10*time.Second)
			}
		}
	}

	if pipeFull {
		store := provider.Store()
		if store == nil {
			return errors.New("Cannot perform action. Messages still loading")
		}
		msg, err := provider.SelectedMessage()
		if err != nil {
			return err
		}
		store.FetchFull([]uint32{msg.Uid}, func(fm *types.FullMessage) {
			if background {
				doExec(fm.Content.Reader)
			} else {
				doTerm(fm.Content.Reader, fmt.Sprintf(
					"%s <%s", cmd[0], msg.Envelope.Subject))
			}
		})
	} else if pipePart {
		p := provider.SelectedMessagePart()
		if p == nil {
			return fmt.Errorf("could not fetch message part")
		}
		store := provider.Store()
		store.FetchBodyPart(p.Msg.Uid, p.Index, func(reader io.Reader) {
			if background {
				doExec(reader)
			} else {
				name := fmt.Sprintf("%s <%s/[%d]",
					cmd[0], p.Msg.Envelope.Subject, p.Index)
				doTerm(reader, name)
			}
		})
	}

	return nil
}

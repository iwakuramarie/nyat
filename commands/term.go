package commands

import (
	"os/exec"

	"github.com/riywo/loginshell"

	"gitea.com/iwakuramarie/nyat/widgets"
)

type Term struct{}

func init() {
	register(Term{})
}

func (Term) Aliases() []string {
	return []string{"terminal", "term"}
}

func (Term) Complete(nyat *widgets.Nyat, args []string) []string {
	return nil
}

// The help command is an alias for `term man` thus Term requires a simple func
func TermCore(nyat *widgets.Nyat, args []string) error {
	if len(args) == 1 {
		shell, err := loginshell.Shell()
		if err != nil {
			return err
		}
		args = append(args, shell)
	}
	term, err := widgets.NewTerminal(exec.Command(args[1], args[2:]...))
	if err != nil {
		return err
	}
	tab := nyat.NewTab(term, args[1])
	term.OnTitle = func(title string) {
		if title == "" {
			title = args[1]
		}
		tab.Name = title
		tab.Content.Invalidate()
	}
	term.OnClose = func(err error) {
		nyat.RemoveTab(term)
		if err != nil {
			nyat.PushError(" " + err.Error())
		}
	}
	return nil
}

func (Term) Execute(nyat *widgets.Nyat, args []string) error {
	return TermCore(nyat, args)
}

package commands

import (
	"errors"

	"gitea.com/iwakuramarie/nyat/widgets"
	"git.sr.ht/~sircmpwn/getopt"
)

type NewAccount struct{}

func init() {
	register(NewAccount{})
}

func (NewAccount) Aliases() []string {
	return []string{"new-account"}
}

func (NewAccount) Complete(nyat *widgets.Nyat, args []string) []string {
	return nil
}

func (NewAccount) Execute(nyat *widgets.Nyat, args []string) error {
	opts, _, err := getopt.Getopts(args, "t")
	if err != nil {
		return errors.New("Usage: new-account [-t]")
	}
	wizard := widgets.NewAccountWizard(nyat.Config(), nyat)
	for _, opt := range opts {
		switch opt.Option {
		case 't':
			wizard.ConfigureTemporaryAccount(true)
		}
	}
	nyat.NewTab(wizard, "New account")
	return nil
}

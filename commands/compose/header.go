package compose

import (
	"fmt"
	"strings"

	"gitea.com/iwakuramarie/nyat/commands"
	"gitea.com/iwakuramarie/nyat/widgets"
	"git.sr.ht/~sircmpwn/getopt"
)

type Header struct{}

var (
	headers = []string{
		"From",
		"To",
		"Cc",
		"Bcc",
		"Subject",
		"Comments",
		"Keywords",
	}
)

func init() {
	register(Header{})
}

func (Header) Aliases() []string {
	return []string{"header"}
}

func (Header) Complete(nyat *widgets.Nyat, args []string) []string {
	return commands.CompletionFromList(headers, args)
}

func (Header) Execute(nyat *widgets.Nyat, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("Usage: %s [-f] field [value]", args[0])
	}

	opts, optind, err := getopt.Getopts(args, "f")
	if err != nil {
		return err
	}
	var (
		force bool = false
	)
	for _, opt := range opts {
		switch opt.Option {
		case 'f':
			force = true
		}
	}

	composer, _ := nyat.SelectedTab().(*widgets.Composer)

	value := strings.Join(args[optind+1:], " ")

	if !force {
		headers, err := composer.PrepareHeader()
		if err != nil {
			return err
		}

		if headers.Has(args[optind]) && value != "" {
			return fmt.Errorf("Header %s already exists", args[optind])
		}
	}

	composer.AddEditor(args[optind], value, false)

	return nil
}

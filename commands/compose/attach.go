package compose

import (
	"fmt"
	"os"
	"strings"

	"gitea.com/iwakuramarie/nyat/commands"
	"gitea.com/iwakuramarie/nyat/widgets"
	"github.com/mitchellh/go-homedir"
)

type Attach struct{}

func init() {
	register(Attach{})
}

func (Attach) Aliases() []string {
	return []string{"attach"}
}

func (Attach) Complete(nyat *widgets.Nyat, args []string) []string {
	path := strings.Join(args, " ")
	return commands.CompletePath(path)
}

func (Attach) Execute(nyat *widgets.Nyat, args []string) error {
	if len(args) == 1 {
		return fmt.Errorf("Usage: :attach <path>")
	}

	path := strings.Join(args[1:], " ")

	path, err := homedir.Expand(path)
	if err != nil {
		nyat.PushError(" " + err.Error())
		return err
	}

	pathinfo, err := os.Stat(path)
	if err != nil {
		nyat.PushError(" " + err.Error())
		return err
	} else if pathinfo.IsDir() {
		nyat.PushError("Attachment must be a file, not a directory")
		return nil
	}

	composer, _ := nyat.SelectedTab().(*widgets.Composer)
	composer.AddAttachment(path)

	nyat.PushSuccess(fmt.Sprintf("Attached %s", pathinfo.Name()))

	return nil
}

package commands

import (
	"errors"
	"os"
	"time"

	"gitea.com/iwakuramarie/nyat/widgets"
)

type PrintWorkDir struct{}

func init() {
	register(PrintWorkDir{})
}

func (PrintWorkDir) Aliases() []string {
	return []string{"pwd"}
}

func (PrintWorkDir) Complete(nyat *widgets.Nyat, args []string) []string {
	return nil
}

func (PrintWorkDir) Execute(nyat *widgets.Nyat, args []string) error {
	if len(args) != 1 {
		return errors.New("Usage: pwd")
	}
	pwd, err := os.Getwd()
	if err != nil {
		return err
	}
	nyat.PushStatus(pwd, 10*time.Second)
	return nil
}

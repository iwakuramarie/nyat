package commands

import (
	"errors"
	"strings"

	"gitea.com/iwakuramarie/nyat/widgets"

	"github.com/go-ini/ini"
)

type Set struct{}

func setUsage() string {
	return "set <category>.<option> <value>"
}

func init() {
	register(Set{})
}

func (Set) Aliases() []string {
	return []string{"set"}

}

func (Set) Complete(nyat *widgets.Nyat, args []string) []string {
	return nil
}

func SetCore(nyat *widgets.Nyat, args []string) error {
	if len(args) != 3 {
		return errors.New("Usage: " + setUsage())
	}

	config := nyat.Config()

	parameters := strings.Split(args[1], ".")

	if len(parameters) != 2 {
		return errors.New("Usage: " + setUsage())
	}

	category := parameters[0]
	option := parameters[1]
	value := args[2]

	new_file := ini.Empty()

	section, err := new_file.NewSection(category)

	if err != nil {
		return nil
	}

	if _, err := section.NewKey(option, value); err != nil {
		return err
	}

	if err := config.LoadConfig(new_file); err != nil {
		return err
	}

	// ensure any ui changes take effect
	nyat.Invalidate()

	return nil
}

func (Set) Execute(nyat *widgets.Nyat, args []string) error {
	return SetCore(nyat, args)
}

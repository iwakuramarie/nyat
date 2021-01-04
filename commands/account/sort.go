package account

import (
	"errors"
	"strings"

	"gitea.com/iwakuramarie/nyat/lib/sort"
	"gitea.com/iwakuramarie/nyat/widgets"
)

type Sort struct{}

func init() {
	register(Sort{})
}

func (Sort) Aliases() []string {
	return []string{"sort"}
}

func (Sort) Complete(nyat *widgets.Nyat, args []string) []string {
	supportedCriteria := []string{
		"arrival",
		"cc",
		"date",
		"from",
		"read",
		"size",
		"subject",
		"to",
	}
	if len(args) == 0 {
		return supportedCriteria
	}
	last := args[len(args)-1]
	var completions []string
	currentPrefix := strings.Join(args, " ") + " "
	// if there is a completed criteria then suggest all again or an option
	for _, criteria := range append(supportedCriteria, "-r") {
		if criteria == last {
			for _, criteria := range supportedCriteria {
				completions = append(completions, currentPrefix+criteria)
			}
			return completions
		}
	}

	currentPrefix = strings.Join(args[:len(args)-1], " ")
	if len(args) > 1 {
		currentPrefix += " "
	}
	// last was beginning an option
	if last == "-" {
		return []string{currentPrefix + "-r"}
	}
	// the last item is not complete
	for _, criteria := range supportedCriteria {
		if strings.HasPrefix(criteria, last) {
			completions = append(completions, currentPrefix+criteria)
		}
	}
	return completions
}

func (Sort) Execute(nyat *widgets.Nyat, args []string) error {
	acct := nyat.SelectedAccount()
	if acct == nil {
		return errors.New("No account selected.")
	}
	store := acct.Store()
	if store == nil {
		return errors.New("Messages still loading.")
	}

	sortCriteria, err := sort.GetSortCriteria(args[1:])
	if err != nil {
		return err
	}

	nyat.SetStatus("Sorting")
	store.Sort(sortCriteria, func() {
		nyat.SetStatus("Sorting complete")
	})
	return nil
}

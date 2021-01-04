package commands

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"gitea.com/iwakuramarie/nyat/widgets"
)

type ChangeTab struct{}

func init() {
	register(ChangeTab{})
}

func (ChangeTab) Aliases() []string {
	return []string{"ct", "change-tab"}
}

func (ChangeTab) Complete(nyat *widgets.Nyat, args []string) []string {
	if len(args) == 0 {
		return nyat.TabNames()
	}
	joinedArgs := strings.Join(args, " ")
	out := make([]string, 0)
	for _, tab := range nyat.TabNames() {
		if strings.HasPrefix(tab, joinedArgs) {
			out = append(out, tab)
		}
	}
	return out
}

func (ChangeTab) Execute(nyat *widgets.Nyat, args []string) error {
	if len(args) == 1 {
		return fmt.Errorf("Usage: %s <tab>", args[0])
	}
	joinedArgs := strings.Join(args[1:], " ")
	if joinedArgs == "-" {
		ok := nyat.SelectPreviousTab()
		if !ok {
			return errors.New("No previous tab to return to")
		}
	} else {
		n, err := strconv.Atoi(joinedArgs)
		if err == nil {
			if strings.HasPrefix(joinedArgs, "+") {
				for ; n > 0; n-- {
					nyat.NextTab()
				}
			} else if strings.HasPrefix(joinedArgs, "-") {
				for ; n < 0; n++ {
					nyat.PrevTab()
				}
			} else {
				ok := nyat.SelectTabIndex(n)
				if !ok {
					return errors.New(
						"No tab with that index")
				}
			}
		} else {
			ok := nyat.SelectTab(joinedArgs)
			if !ok {
				return errors.New("No tab with that name")
			}
		}
	}
	return nil
}

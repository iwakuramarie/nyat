package account

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"gitea.com/iwakuramarie/nyat/widgets"
)

type NextPrevMsg struct{}

func init() {
	register(NextPrevMsg{})
}

func (NextPrevMsg) Aliases() []string {
	return []string{"next", "next-message", "prev", "prev-message"}
}

func (NextPrevMsg) Complete(nyat *widgets.Nyat, args []string) []string {
	return nil
}

func (NextPrevMsg) Execute(nyat *widgets.Nyat, args []string) error {
	n, pct, err := ParseNextPrevMessage(args)
	if err != nil {
		return err
	}
	acct := nyat.SelectedAccount()
	if acct == nil {
		return errors.New("No account selected")
	}
	return ExecuteNextPrevMessage(args, acct, pct, n)
}

func ParseNextPrevMessage(args []string) (int, bool, error) {
	if len(args) > 2 {
		return 0, false, nextPrevMessageUsage(args[0])
	}
	var (
		n   int = 1
		err error
		pct bool
	)
	if len(args) > 1 {
		if strings.HasSuffix(args[1], "%") {
			pct = true
			args[1] = args[1][:len(args[1])-1]
		}
		n, err = strconv.Atoi(args[1])
		if err != nil {
			return 0, false, nextPrevMessageUsage(args[0])
		}
	}
	return n, pct, nil
}

func ExecuteNextPrevMessage(args []string, acct *widgets.AccountView, pct bool, n int) error {
	if pct {
		n = int(float64(acct.Messages().Height()) * (float64(n) / 100.0))
	}
	if args[0] == "prev-message" || args[0] == "prev" {
		store := acct.Store()
		if store != nil {
			store.NextPrev(-n)
			acct.Messages().Invalidate()
		}
	} else {
		store := acct.Store()
		if store != nil {
			store.NextPrev(n)
			acct.Messages().Invalidate()
		}
	}
	return nil
}

func nextPrevMessageUsage(cmd string) error {
	return fmt.Errorf("Usage: %s [<n>[%%]]", cmd)
}

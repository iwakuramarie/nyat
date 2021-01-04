package account

import (
	"errors"

	"gitea.com/iwakuramarie/nyat/widgets"
)

type SearchFilter struct{}

func init() {
	register(SearchFilter{})
}

func (SearchFilter) Aliases() []string {
	return []string{"search", "filter"}
}

func (SearchFilter) Complete(nyat *widgets.Nyat, args []string) []string {
	return nil
}

func (SearchFilter) Execute(nyat *widgets.Nyat, args []string) error {
	acct := nyat.SelectedAccount()
	if acct == nil {
		return errors.New("No account selected")
	}
	store := acct.Store()
	if store == nil {
		return errors.New("Cannot perform action. Messages still loading")
	}

	var cb func([]uint32)
	if args[0] == "filter" {
		nyat.SetStatus("Filtering...")
		cb = func(uids []uint32) {
			nyat.SetStatus("Filter complete.")
			acct.Logger().Printf("Filter results: %v", uids)
			store.ApplyFilter(uids)
		}
	} else {
		nyat.SetStatus("Searching...")
		cb = func(uids []uint32) {
			nyat.SetStatus("Search complete.")
			acct.Logger().Printf("Search results: %v", uids)
			store.ApplySearch(uids)
			// TODO: Remove when stores have multiple OnUpdate handlers
			acct.Messages().Invalidate()
		}
	}
	store.Search(args, cb)
	return nil
}

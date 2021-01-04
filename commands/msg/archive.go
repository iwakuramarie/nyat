package msg

import (
	"errors"
	"fmt"
	"path"
	"sync"
	"time"

	"gitea.com/iwakuramarie/nyat/commands"
	"gitea.com/iwakuramarie/nyat/models"
	"gitea.com/iwakuramarie/nyat/widgets"
	"gitea.com/iwakuramarie/nyat/worker/types"
)

const (
	ARCHIVE_FLAT  = "flat"
	ARCHIVE_YEAR  = "year"
	ARCHIVE_MONTH = "month"
)

type Archive struct{}

func init() {
	register(Archive{})
}

func (Archive) Aliases() []string {
	return []string{"archive"}
}

func (Archive) Complete(nyat *widgets.Nyat, args []string) []string {
	valid := []string{"flat", "year", "month"}
	return commands.CompletionFromList(valid, args)
}

func (Archive) Execute(nyat *widgets.Nyat, args []string) error {
	if len(args) != 2 {
		return errors.New("Usage: archive <flat|year|month>")
	}
	h := newHelper(nyat)
	acct, err := h.account()
	if err != nil {
		return err
	}
	store, err := h.store()
	if err != nil {
		return err
	}
	msgs, err := h.messages()
	if err != nil {
		return err
	}
	archiveDir := acct.AccountConfig().Archive
	store.Next()
	acct.Messages().Invalidate()

	var uidMap map[string][]uint32
	switch args[1] {
	case ARCHIVE_MONTH:
		uidMap = groupBy(msgs, func(msg *models.MessageInfo) string {
			dir := path.Join(archiveDir,
				fmt.Sprintf("%d", msg.Envelope.Date.Year()),
				fmt.Sprintf("%02d", msg.Envelope.Date.Month()))
			return dir
		})
	case ARCHIVE_YEAR:
		uidMap = groupBy(msgs, func(msg *models.MessageInfo) string {
			dir := path.Join(archiveDir, fmt.Sprintf("%v",
				msg.Envelope.Date.Year()))
			return dir
		})
	case ARCHIVE_FLAT:
		uidMap = make(map[string][]uint32)
		uidMap[archiveDir] = commands.UidsFromMessageInfos(msgs)
	}

	var wg sync.WaitGroup
	wg.Add(len(uidMap))
	success := true

	for dir, uids := range uidMap {
		store.Move(uids, dir, true, func(
			msg types.WorkerMessage) {
			switch msg := msg.(type) {
			case *types.Done:
				wg.Done()
			case *types.Error:
				nyat.PushError(" " + msg.Error.Error())
				success = false
				wg.Done()
			}
		})
	}
	// we need to do that in the background, else we block the main thread
	go func() {
		wg.Wait()
		if success {
			nyat.PushStatus("Messages archived.", 10*time.Second)
		}
	}()
	return nil
}

func groupBy(msgs []*models.MessageInfo,
	grouper func(*models.MessageInfo) string) map[string][]uint32 {
	m := make(map[string][]uint32)
	for _, msg := range msgs {
		group := grouper(msg)
		m[group] = append(m[group], msg.Uid)
	}
	return m
}

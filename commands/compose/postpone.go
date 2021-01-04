package compose

import (
	"io"
	"io/ioutil"
	"time"

	"github.com/miolini/datacounter"
	"github.com/pkg/errors"

	"gitea.com/iwakuramarie/nyat/models"
	"gitea.com/iwakuramarie/nyat/widgets"
	"gitea.com/iwakuramarie/nyat/worker/types"
)

type Postpone struct{}

func init() {
	register(Postpone{})
}

func (Postpone) Aliases() []string {
	return []string{"postpone"}
}

func (Postpone) Complete(nyat *widgets.Nyat, args []string) []string {
	return nil
}

func (Postpone) Execute(nyat *widgets.Nyat, args []string) error {
	if len(args) != 1 {
		return errors.New("Usage: postpone")
	}
	composer, _ := nyat.SelectedTab().(*widgets.Composer)
	config := composer.Config()

	if config.Postpone == "" {
		return errors.New("No Postpone location configured")
	}

	nyat.Logger().Println("Postponing mail")

	header, err := composer.PrepareHeader()
	if err != nil {
		return errors.Wrap(err, "PrepareHeader")
	}
	header.SetContentType("text/plain", map[string]string{"charset": "UTF-8"})
	header.Set("Content-Transfer-Encoding", "quoted-printable")
	worker := composer.Worker()
	dirs := nyat.SelectedAccount().Directories().List()
	alreadyCreated := false
	for _, dir := range dirs {
		if dir == config.Postpone {
			alreadyCreated = true
			break
		}
	}

	errChan := make(chan string)

	// run this as a goroutine so we can make other progress. The message
	// will be saved once the directory is created.
	go func() {
		errStr := <-errChan
		if errStr != "" {
			nyat.PushError(" " + errStr)
			return
		}

		nyat.RemoveTab(composer)
		ctr := datacounter.NewWriterCounter(ioutil.Discard)
		err = composer.WriteMessage(header, ctr)
		if err != nil {
			nyat.PushError(errors.Wrap(err, "WriteMessage").Error())
			composer.Close()
			return
		}
		nbytes := int(ctr.Count())
		r, w := io.Pipe()
		worker.PostAction(&types.AppendMessage{
			Destination: config.Postpone,
			Flags:       []models.Flag{models.SeenFlag},
			Date:        time.Now(),
			Reader:      r,
			Length:      int(nbytes),
		}, func(msg types.WorkerMessage) {
			switch msg := msg.(type) {
			case *types.Done:
				nyat.PushStatus("Message postponed.", 10*time.Second)
				r.Close()
				composer.Close()
			case *types.Error:
				nyat.PushError(" " + msg.Error.Error())
				r.Close()
				composer.Close()
			}
		})
		composer.WriteMessage(header, w)
		w.Close()
	}()

	if !alreadyCreated {
		// to synchronise the creating of the directory
		worker.PostAction(&types.CreateDirectory{
			Directory: config.Postpone,
		}, func(msg types.WorkerMessage) {
			switch msg := msg.(type) {
			case *types.Done:
				errChan <- ""
			case *types.Error:
				errChan <- msg.Error.Error()
			}
		})
	} else {
		errChan <- ""
	}

	return nil
}

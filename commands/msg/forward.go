package msg

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"strings"

	"gitea.com/iwakuramarie/nyat/lib"
	"gitea.com/iwakuramarie/nyat/lib/format"
	"gitea.com/iwakuramarie/nyat/models"
	"gitea.com/iwakuramarie/nyat/widgets"
	"gitea.com/iwakuramarie/nyat/worker/types"
	"github.com/emersion/go-message/mail"

	"git.sr.ht/~sircmpwn/getopt"
)

type forward struct{}

func init() {
	register(forward{})
}

func (forward) Aliases() []string {
	return []string{"forward"}
}

func (forward) Complete(nyat *widgets.Nyat, args []string) []string {
	return nil
}

func (forward) Execute(nyat *widgets.Nyat, args []string) error {
	opts, optind, err := getopt.Getopts(args, "AT:")
	if err != nil {
		return err
	}
	attach := false
	template := ""
	for _, opt := range opts {
		switch opt.Option {
		case 'A':
			attach = true
		case 'T':
			template = opt.Value
		}
	}

	widget := nyat.SelectedTab().(widgets.ProvidesMessage)
	acct := widget.SelectedAccount()
	if acct == nil {
		return errors.New("No account selected")
	}
	store := widget.Store()
	if store == nil {
		return errors.New("Cannot perform action. Messages still loading")
	}
	msg, err := widget.SelectedMessage()
	if err != nil {
		return err
	}
	acct.Logger().Println("Forwarding email " + msg.Envelope.MessageId)

	h := &mail.Header{}
	subject := "Fwd: " + msg.Envelope.Subject
	h.SetSubject(subject)

	if len(args) != 1 {
		to := strings.Join(args[optind:], ", ")
		tolist, err := mail.ParseAddressList(to)
		if err != nil {
			return fmt.Errorf("invalid to address(es): %v", err)
		}
		h.SetAddressList("to", tolist)
	}

	original := models.OriginalMail{
		From:          format.FormatAddresses(msg.Envelope.From),
		Date:          msg.Envelope.Date,
		RFC822Headers: msg.RFC822Headers,
	}

	addTab := func() (*widgets.Composer, error) {
		composer, err := widgets.NewComposer(nyat, acct, nyat.Config(),
			acct.AccountConfig(), acct.Worker(), template, h, original)
		if err != nil {
			nyat.PushError("Error: " + err.Error())
			return nil, err
		}

		tab := nyat.NewTab(composer, subject)
		if !h.Has("to") {
			composer.FocusRecipient()
		} else {
			composer.FocusTerminal()
		}
		composer.OnHeaderChange("Subject", func(subject string) {
			if subject == "" {
				tab.Name = "New email"
			} else {
				tab.Name = subject
			}
			tab.Content.Invalidate()
		})
		return composer, nil
	}

	if attach {
		tmpDir, err := ioutil.TempDir("", "nyat-tmp-attachment")
		if err != nil {
			return err
		}
		tmpFileName := path.Join(tmpDir,
			strings.ReplaceAll(fmt.Sprintf("%s.eml", msg.Envelope.Subject), "/", "-"))
		store.FetchFull([]uint32{msg.Uid}, func(fm *types.FullMessage) {
			tmpFile, err := os.Create(tmpFileName)
			if err != nil {
				println(err)
				// TODO: Do something with the error
				addTab()
				return
			}

			defer tmpFile.Close()
			io.Copy(tmpFile, fm.Content.Reader)
			composer, err := addTab()
			if err != nil {
				return
			}
			composer.AddAttachment(tmpFileName)
			composer.OnClose(func(composer *widgets.Composer) {
				os.RemoveAll(tmpDir)
			})
		})
	} else {
		if template == "" {
			template = nyat.Config().Templates.Forwards
		}

		// TODO: add attachments!
		part := lib.FindPlaintext(msg.BodyStructure, nil)
		if part == nil {
			part = lib.FindFirstNonMultipart(msg.BodyStructure, nil)
			// if it's still nil here, we don't have a multipart msg, that's fine
		}
		store.FetchBodyPart(msg.Uid, part, func(reader io.Reader) {
			buf := new(bytes.Buffer)
			buf.ReadFrom(reader)
			original.Text = buf.String()
			addTab()
		})
	}
	return nil
}

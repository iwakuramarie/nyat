package msgview

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"mime"
	"os"
	"time"

	"gitea.com/iwakuramarie/nyat/lib"
	"gitea.com/iwakuramarie/nyat/widgets"
)

type Open struct{}

func init() {
	register(Open{})
}

func (Open) Aliases() []string {
	return []string{"open"}
}

func (Open) Complete(nyat *widgets.Nyat, args []string) []string {
	return nil
}

func (Open) Execute(nyat *widgets.Nyat, args []string) error {
	if len(args) != 1 {
		return errors.New("Usage: open")
	}

	mv := nyat.SelectedTab().(*widgets.MessageViewer)
	p := mv.SelectedMessagePart()

	store := mv.Store()
	store.FetchBodyPart(p.Msg.Uid, p.Index, func(reader io.Reader) {
		extension := ""
		// try to determine the correct extension based on mimetype
		if part, err := p.Msg.BodyStructure.PartAtIndex(p.Index); err == nil {
			mimeType := fmt.Sprintf("%s/%s", part.MIMEType, part.MIMESubType)

			if exts, _ := mime.ExtensionsByType(mimeType); exts != nil && len(exts) > 0 {
				extension = exts[0]
			}
		}

		tmpFile, err := ioutil.TempFile(os.TempDir(), "nyat-*"+extension)
		if err != nil {
			nyat.PushError(" " + err.Error())
			return
		}
		defer tmpFile.Close()

		_, err = io.Copy(tmpFile, reader)
		if err != nil {
			nyat.PushError(" " + err.Error())
			return
		}

		lib.OpenFile(tmpFile.Name(), func(err error) {
			nyat.PushError(" " + err.Error())
		})

		nyat.PushStatus("Opened", 10*time.Second)
	})

	return nil
}

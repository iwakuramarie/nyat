package imap

import (
	"crypto/tls"
	"fmt"
	"net/url"
	"strings"

	"github.com/emersion/go-imap"
	idle "github.com/emersion/go-imap-idle"
	sortthread "github.com/emersion/go-imap-sortthread"
	"github.com/emersion/go-imap/client"
	"golang.org/x/oauth2"

	"gitea.com/iwakuramarie/nyat/lib"
	"gitea.com/iwakuramarie/nyat/models"
	"gitea.com/iwakuramarie/nyat/worker/handlers"
	"gitea.com/iwakuramarie/nyat/worker/types"
)

func init() {
	handlers.RegisterWorkerFactory("imap", NewIMAPWorker)
	handlers.RegisterWorkerFactory("imaps", NewIMAPWorker)
}

var errUnsupported = fmt.Errorf("unsupported command")

type imapClient struct {
	*client.Client
	idle *idle.IdleClient
	sort *sortthread.SortClient
}

type IMAPWorker struct {
	config struct {
		scheme      string
		insecure    bool
		addr        string
		user        *url.Userinfo
		folders     []string
		oauthBearer lib.OAuthBearer
	}

	client   *imapClient
	idleStop chan struct{}
	idleDone chan error
	selected *imap.MailboxStatus
	updates  chan client.Update
	worker   *types.Worker
	// Map of sequence numbers to UIDs, index 0 is seq number 1
	seqMap []uint32
}

func NewIMAPWorker(worker *types.Worker) (types.Backend, error) {
	return &IMAPWorker{
		idleDone: make(chan error),
		updates:  make(chan client.Update, 50),
		worker:   worker,
		selected: &imap.MailboxStatus{},
	}, nil
}

func (w *IMAPWorker) handleMessage(msg types.WorkerMessage) error {
	if w.idleStop != nil {
		close(w.idleStop)
		if err := <-w.idleDone; err != nil {
			w.worker.PostMessage(&types.Error{Error: err}, nil)
		}
	}

	var reterr error // will be returned at the end, needed to support idle

	switch msg := msg.(type) {
	case *types.Unsupported:
		// No-op
	case *types.Configure:
		u, err := url.Parse(msg.Config.Source)
		if err != nil {
			return err
		}

		w.config.scheme = u.Scheme
		if strings.HasSuffix(w.config.scheme, "+insecure") {
			w.config.scheme = strings.TrimSuffix(w.config.scheme, "+insecure")
			w.config.insecure = true
		}

		if strings.HasSuffix(w.config.scheme, "+oauthbearer") {
			w.config.scheme = strings.TrimSuffix(w.config.scheme, "+oauthbearer")
			w.config.oauthBearer.Enabled = true
			q := u.Query()

			oauth2 := &oauth2.Config{}
			if q.Get("token_endpoint") != "" {
				oauth2.ClientID = q.Get("client_id")
				oauth2.ClientSecret = q.Get("client_secret")
				oauth2.Scopes = []string{q.Get("scope")}
				oauth2.Endpoint.TokenURL = q.Get("token_endpoint")
			}
			w.config.oauthBearer.OAuth2 = oauth2
		}

		w.config.addr = u.Host
		if !strings.ContainsRune(w.config.addr, ':') {
			w.config.addr += ":" + w.config.scheme
		}

		w.config.user = u.User
		w.config.folders = msg.Config.Folders
	case *types.Connect:
		var (
			c   *client.Client
			err error
		)
		switch w.config.scheme {
		case "imap":
			c, err = client.Dial(w.config.addr)
			if err != nil {
				return err
			}

			if !w.config.insecure {
				if err := c.StartTLS(&tls.Config{}); err != nil {
					return err
				}
			}
		case "imaps":
			c, err = client.DialTLS(w.config.addr, &tls.Config{})
			if err != nil {
				return err
			}
		default:
			return fmt.Errorf("Unknown IMAP scheme %s", w.config.scheme)
		}
		c.ErrorLog = w.worker.Logger

		if w.config.user != nil {
			username := w.config.user.Username()
			password, hasPassword := w.config.user.Password()
			if !hasPassword {
				// TODO: ask password
			}

			if w.config.oauthBearer.Enabled {
				if err := w.config.oauthBearer.Authenticate(username, password, c); err != nil {
					return err
				}
			} else if err := c.Login(username, password); err != nil {
				return err
			}
		}

		c.SetDebug(w.worker.Logger.Writer())

		if _, err := c.Select(imap.InboxName, false); err != nil {
			return err
		}

		c.Updates = w.updates
		w.client = &imapClient{c, idle.NewClient(c), sortthread.NewSortClient(c)}
		w.worker.PostMessage(&types.Done{types.RespondTo(msg)}, nil)
	case *types.ListDirectories:
		w.handleListDirectories(msg)
	case *types.OpenDirectory:
		w.handleOpenDirectory(msg)
	case *types.FetchDirectoryContents:
		w.handleFetchDirectoryContents(msg)
	case *types.CreateDirectory:
		w.handleCreateDirectory(msg)
	case *types.RemoveDirectory:
		w.handleRemoveDirectory(msg)
	case *types.FetchMessageHeaders:
		w.handleFetchMessageHeaders(msg)
	case *types.FetchMessageBodyPart:
		w.handleFetchMessageBodyPart(msg)
	case *types.FetchFullMessages:
		w.handleFetchFullMessages(msg)
	case *types.DeleteMessages:
		w.handleDeleteMessages(msg)
	case *types.FlagMessages:
		w.handleFlagMessages(msg)
	case *types.AnsweredMessages:
		w.handleAnsweredMessages(msg)
	case *types.CopyMessages:
		w.handleCopyMessages(msg)
	case *types.AppendMessage:
		w.handleAppendMessage(msg)
	case *types.SearchDirectory:
		w.handleSearchDirectory(msg)
	default:
		reterr = errUnsupported
	}

	if w.idleStop != nil {
		w.idleStop = make(chan struct{})
		go func() {
			w.idleDone <- w.client.idle.IdleWithFallback(w.idleStop, 0)
		}()
	}
	return reterr
}

func (w *IMAPWorker) handleImapUpdate(update client.Update) {
	w.worker.Logger.Printf("(= %T", update)
	switch update := update.(type) {
	case *client.MailboxUpdate:
		status := update.Mailbox
		if w.selected.Name == status.Name {
			w.selected = status
		}
		w.worker.PostMessage(&types.DirectoryInfo{
			Info: &models.DirectoryInfo{
				Flags:    status.Flags,
				Name:     status.Name,
				ReadOnly: status.ReadOnly,

				Exists: int(status.Messages),
				Recent: int(status.Recent),
				Unseen: int(status.Unseen),
			},
		}, nil)
	case *client.MessageUpdate:
		msg := update.Message
		if msg.Uid == 0 {
			msg.Uid = w.seqMap[msg.SeqNum-1]
		}
		w.worker.PostMessage(&types.MessageInfo{
			Info: &models.MessageInfo{
				BodyStructure: translateBodyStructure(msg.BodyStructure),
				Envelope:      translateEnvelope(msg.Envelope),
				Flags:         translateImapFlags(msg.Flags),
				InternalDate:  msg.InternalDate,
				Uid:           msg.Uid,
			},
		}, nil)
	case *client.ExpungeUpdate:
		i := update.SeqNum - 1
		uid := w.seqMap[i]
		w.seqMap = append(w.seqMap[:i], w.seqMap[i+1:]...)
		w.worker.PostMessage(&types.MessagesDeleted{
			Uids: []uint32{uid},
		}, nil)
	}
}

func (w *IMAPWorker) Run() {
	for {
		select {
		case msg := <-w.worker.Actions:
			msg = w.worker.ProcessAction(msg)
			if err := w.handleMessage(msg); err == errUnsupported {
				w.worker.PostMessage(&types.Unsupported{
					Message: types.RespondTo(msg),
				}, nil)
			} else if err != nil {
				w.worker.PostMessage(&types.Error{
					Message: types.RespondTo(msg),
					Error:   err,
				}, nil)
			}
		case update := <-w.updates:
			w.handleImapUpdate(update)
		}
	}
}

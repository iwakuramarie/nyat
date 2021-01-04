package widgets

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net/url"
	"strings"
	"time"

	"github.com/emersion/go-message/mail"
	"github.com/gdamore/tcell/v2"
	"github.com/google/shlex"
	"golang.org/x/crypto/openpgp"

	"gitea.com/iwakuramarie/nyat/config"
	"gitea.com/iwakuramarie/nyat/lib"
	"gitea.com/iwakuramarie/nyat/lib/ui"
	"gitea.com/iwakuramarie/nyat/models"
)

type Nyat struct {
	accounts    map[string]*AccountView
	cmd         func(cmd []string) error
	cmdHistory  lib.History
	complete    func(cmd string) []string
	conf        *config.NyatConfig
	focused     ui.Interactive
	grid        *ui.Grid
	logger      *log.Logger
	simulating  int
	statusbar   *ui.Stack
	statusline  *StatusLine
	pendingKeys []config.KeyStroke
	prompts     *ui.Stack
	tabs        *ui.Tabs
	ui          *ui.UI
	beep        func() error
	dialog      ui.DrawableInteractive
}

type Choice struct {
	Key     string
	Text    string
	Command []string
}

func NewNyat(conf *config.NyatConfig, logger *log.Logger,
	cmd func(cmd []string) error, complete func(cmd string) []string,
	cmdHistory lib.History) *Nyat {

	tabs := ui.NewTabs(&conf.Ui)

	statusbar := ui.NewStack(conf.Ui)
	statusline := NewStatusLine(conf.Ui)
	statusbar.Push(statusline)

	grid := ui.NewGrid().Rows([]ui.GridSpec{
		{ui.SIZE_EXACT, ui.Const(1)},
		{ui.SIZE_WEIGHT, ui.Const(1)},
		{ui.SIZE_EXACT, ui.Const(1)},
	}).Columns([]ui.GridSpec{
		{ui.SIZE_WEIGHT, ui.Const(1)},
	})
	grid.AddChild(tabs.TabStrip)
	grid.AddChild(tabs.TabContent).At(1, 0)
	grid.AddChild(statusbar).At(2, 0)

	nyat := &Nyat{
		accounts:   make(map[string]*AccountView),
		conf:       conf,
		cmd:        cmd,
		cmdHistory: cmdHistory,
		complete:   complete,
		grid:       grid,
		logger:     logger,
		statusbar:  statusbar,
		statusline: statusline,
		prompts:    ui.NewStack(conf.Ui),
		tabs:       tabs,
	}

	statusline.SetNyat(nyat)
	conf.Triggers.ExecuteCommand = cmd

	for i, acct := range conf.Accounts {
		view, err := NewAccountView(nyat, conf, &conf.Accounts[i], logger, nyat)
		if err != nil {
			tabs.Add(errorScreen(err.Error(), conf.Ui), acct.Name)
		} else {
			nyat.accounts[acct.Name] = view
			tabs.Add(view, acct.Name)
		}
	}

	if len(conf.Accounts) == 0 {
		wizard := NewAccountWizard(nyat.Config(), nyat)
		wizard.Focus(true)
		nyat.NewTab(wizard, "New account")
	}

	tabs.CloseTab = func(index int) {
		switch content := nyat.tabs.Tabs[index].Content.(type) {
		case *AccountView:
			return
		case *AccountWizard:
			return
		case *Composer:
			nyat.RemoveTab(content)
			content.Close()
		case *Terminal:
			content.Close(nil)
		case *MessageViewer:
			nyat.RemoveTab(content)
		}
	}

	return nyat
}

func (nyat *Nyat) OnBeep(f func() error) {
	nyat.beep = f
}

func (nyat *Nyat) Beep() {
	if nyat.beep == nil {
		nyat.logger.Printf("should beep, but no beeper")
		return
	}
	if err := nyat.beep(); err != nil {
		nyat.logger.Printf("tried to beep, but could not: %v", err)
	}
}

func (nyat *Nyat) Tick() bool {
	more := false
	for _, acct := range nyat.accounts {
		more = acct.Tick() || more
	}

	if len(nyat.prompts.Children()) > 0 {
		more = true
		previous := nyat.focused
		prompt := nyat.prompts.Pop().(*ExLine)
		prompt.finish = func() {
			nyat.statusbar.Pop()
			nyat.focus(previous)
		}

		nyat.statusbar.Push(prompt)
		nyat.focus(prompt)
	}

	return more
}

func (nyat *Nyat) Children() []ui.Drawable {
	return nyat.grid.Children()
}

func (nyat *Nyat) OnInvalidate(onInvalidate func(d ui.Drawable)) {
	nyat.grid.OnInvalidate(func(_ ui.Drawable) {
		onInvalidate(nyat)
	})
}

func (nyat *Nyat) Invalidate() {
	nyat.grid.Invalidate()
}

func (nyat *Nyat) Focus(focus bool) {
	// who cares
}

func (nyat *Nyat) Draw(ctx *ui.Context) {
	nyat.grid.Draw(ctx)
	if nyat.dialog != nil {
		nyat.dialog.Draw(ctx.Subcontext(4, ctx.Height()/2-2,
			ctx.Width()-8, 4))
	}
}

func (nyat *Nyat) getBindings() *config.KeyBindings {
	switch view := nyat.SelectedTab().(type) {
	case *AccountView:
		return nyat.conf.Bindings.MessageList
	case *AccountWizard:
		return nyat.conf.Bindings.AccountWizard
	case *Composer:
		switch view.Bindings() {
		case "compose::editor":
			return nyat.conf.Bindings.ComposeEditor
		case "compose::review":
			return nyat.conf.Bindings.ComposeReview
		default:
			return nyat.conf.Bindings.Compose
		}
	case *MessageViewer:
		return nyat.conf.Bindings.MessageView
	case *Terminal:
		return nyat.conf.Bindings.Terminal
	default:
		return nyat.conf.Bindings.Global
	}
}

func (nyat *Nyat) simulate(strokes []config.KeyStroke) {
	nyat.pendingKeys = []config.KeyStroke{}
	nyat.simulating += 1
	for _, stroke := range strokes {
		simulated := tcell.NewEventKey(
			stroke.Key, stroke.Rune, tcell.ModNone)
		nyat.Event(simulated)
	}
	nyat.simulating -= 1
}

func (nyat *Nyat) Event(event tcell.Event) bool {
	if nyat.dialog != nil {
		return nyat.dialog.Event(event)
	}

	if nyat.focused != nil {
		return nyat.focused.Event(event)
	}

	switch event := event.(type) {
	case *tcell.EventKey:
		nyat.statusline.Expire()
		nyat.pendingKeys = append(nyat.pendingKeys, config.KeyStroke{
			Key:  event.Key(),
			Rune: event.Rune(),
		})
		nyat.statusline.Invalidate()
		bindings := nyat.getBindings()
		incomplete := false
		result, strokes := bindings.GetBinding(nyat.pendingKeys)
		switch result {
		case config.BINDING_FOUND:
			nyat.simulate(strokes)
			return true
		case config.BINDING_INCOMPLETE:
			incomplete = true
		case config.BINDING_NOT_FOUND:
		}
		if bindings.Globals {
			result, strokes = nyat.conf.Bindings.Global.
				GetBinding(nyat.pendingKeys)
			switch result {
			case config.BINDING_FOUND:
				nyat.simulate(strokes)
				return true
			case config.BINDING_INCOMPLETE:
				incomplete = true
			case config.BINDING_NOT_FOUND:
			}
		}
		if !incomplete {
			nyat.pendingKeys = []config.KeyStroke{}
			exKey := bindings.ExKey
			if nyat.simulating > 0 {
				// Keybindings still use : even if you change the ex key
				exKey = nyat.conf.Bindings.Global.ExKey
			}
			if event.Key() == exKey.Key && event.Rune() == exKey.Rune {
				nyat.BeginExCommand("")
				return true
			}
			interactive, ok := nyat.tabs.Tabs[nyat.tabs.Selected].Content.(ui.Interactive)
			if ok {
				return interactive.Event(event)
			}
			return false
		}
	case *tcell.EventMouse:
		if event.Buttons() == tcell.ButtonNone {
			return false
		}
		x, y := event.Position()
		nyat.grid.MouseEvent(x, y, event)
		return true
	}
	return false
}

func (nyat *Nyat) Config() *config.NyatConfig {
	return nyat.conf
}

func (nyat *Nyat) Logger() *log.Logger {
	return nyat.logger
}

func (nyat *Nyat) SelectedAccount() *AccountView {
	switch tab := nyat.SelectedTab().(type) {
	case *AccountView:
		return tab
	case *MessageViewer:
		return tab.SelectedAccount()
	case *Composer:
		return tab.Account()
	}
	return nil
}

func (nyat *Nyat) SelectedTab() ui.Drawable {
	return nyat.tabs.Tabs[nyat.tabs.Selected].Content
}

func (nyat *Nyat) SelectedTabIndex() int {
	return nyat.tabs.Selected
}

func (nyat *Nyat) NumTabs() int {
	return len(nyat.tabs.Tabs)
}

func (nyat *Nyat) NewTab(clickable ui.Drawable, name string) *ui.Tab {
	tab := nyat.tabs.Add(clickable, name)
	nyat.tabs.Select(len(nyat.tabs.Tabs) - 1)
	return tab
}

func (nyat *Nyat) RemoveTab(tab ui.Drawable) {
	nyat.tabs.Remove(tab)
}

func (nyat *Nyat) ReplaceTab(tabSrc ui.Drawable, tabTarget ui.Drawable, name string) {
	nyat.tabs.Replace(tabSrc, tabTarget, name)
}

func (nyat *Nyat) MoveTab(i int) {
	nyat.tabs.MoveTab(i)
}

func (nyat *Nyat) PinTab() {
	nyat.tabs.PinTab()
}

func (nyat *Nyat) UnpinTab() {
	nyat.tabs.UnpinTab()
}

func (nyat *Nyat) NextTab() {
	nyat.tabs.NextTab()
}

func (nyat *Nyat) PrevTab() {
	nyat.tabs.PrevTab()
}

func (nyat *Nyat) SelectTab(name string) bool {
	for i, tab := range nyat.tabs.Tabs {
		if tab.Name == name {
			nyat.tabs.Select(i)
			return true
		}
	}
	return false
}

func (nyat *Nyat) SelectTabIndex(index int) bool {
	for i := range nyat.tabs.Tabs {
		if i == index {
			nyat.tabs.Select(i)
			return true
		}
	}
	return false
}

func (nyat *Nyat) TabNames() []string {
	var names []string
	for _, tab := range nyat.tabs.Tabs {
		names = append(names, tab.Name)
	}
	return names
}

func (nyat *Nyat) SelectPreviousTab() bool {
	return nyat.tabs.SelectPrevious()
}

// TODO: Use per-account status lines, but a global ex line
func (nyat *Nyat) SetStatus(status string) *StatusMessage {
	return nyat.statusline.Set(status)
}

func (nyat *Nyat) SetError(status string) *StatusMessage {
	return nyat.statusline.SetError(status)
}

func (nyat *Nyat) PushStatus(text string, expiry time.Duration) *StatusMessage {
	return nyat.statusline.Push(text, expiry)
}

func (nyat *Nyat) PushError(text string) *StatusMessage {
	return nyat.statusline.PushError(text)
}

func (nyat *Nyat) PushSuccess(text string) *StatusMessage {
	return nyat.statusline.PushSuccess(text)
}

func (nyat *Nyat) focus(item ui.Interactive) {
	if nyat.focused == item {
		return
	}
	if nyat.focused != nil {
		nyat.focused.Focus(false)
	}
	nyat.focused = item
	interactive, ok := nyat.tabs.Tabs[nyat.tabs.Selected].Content.(ui.Interactive)
	if item != nil {
		item.Focus(true)
		if ok {
			interactive.Focus(false)
		}
	} else {
		if ok {
			interactive.Focus(true)
		}
	}
}

func (nyat *Nyat) BeginExCommand(cmd string) {
	previous := nyat.focused
	exline := NewExLine(nyat.conf, cmd, func(cmd string) {
		parts, err := shlex.Split(cmd)
		if err != nil {
			nyat.PushError(" " + err.Error())
		}
		err = nyat.cmd(parts)
		if err != nil {
			nyat.PushError(" " + err.Error())
		}
		// only add to history if this is an unsimulated command,
		// ie one not executed from a keybinding
		if nyat.simulating == 0 {
			nyat.cmdHistory.Add(cmd)
		}
	}, func() {
		nyat.statusbar.Pop()
		nyat.focus(previous)
	}, func(cmd string) []string {
		return nyat.complete(cmd)
	}, nyat.cmdHistory)
	nyat.statusbar.Push(exline)
	nyat.focus(exline)
}

func (nyat *Nyat) RegisterPrompt(prompt string, cmd []string) {
	p := NewPrompt(nyat.conf, prompt, func(text string) {
		if text != "" {
			cmd = append(cmd, text)
		}
		err := nyat.cmd(cmd)
		if err != nil {
			nyat.PushError(" " + err.Error())
		}
	}, func(cmd string) []string {
		return nil // TODO: completions
	})
	nyat.prompts.Push(p)
}

func (nyat *Nyat) RegisterChoices(choices []Choice) {
	cmds := make(map[string][]string)
	texts := []string{}
	for _, c := range choices {
		text := fmt.Sprintf("[%s] %s", c.Key, c.Text)
		if strings.Contains(c.Text, c.Key) {
			text = strings.Replace(c.Text, c.Key, "["+c.Key+"]", 1)
		}
		texts = append(texts, text)
		cmds[c.Key] = c.Command
	}
	prompt := strings.Join(texts, ", ") + "? "
	p := NewPrompt(nyat.conf, prompt, func(text string) {
		cmd, ok := cmds[text]
		if !ok {
			return
		}
		err := nyat.cmd(cmd)
		if err != nil {
			nyat.PushError(" " + err.Error())
		}
	}, func(cmd string) []string {
		return nil // TODO: completions
	})
	nyat.prompts.Push(p)
}

func (nyat *Nyat) Mailto(addr *url.URL) error {
	acct := nyat.SelectedAccount()
	if acct == nil {
		return errors.New("No account selected")
	}

	var subject string
	h := &mail.Header{}
	to, err := mail.ParseAddressList(addr.Opaque)
	if err != nil {
		return fmt.Errorf("Could not parse to: %v", err)
	}
	h.SetAddressList("to", to)
	for key, vals := range addr.Query() {
		switch strings.ToLower(key) {
		case "cc":
			list, err := mail.ParseAddressList(strings.Join(vals, ","))
			if err != nil {
				break
			}
			h.SetAddressList("Cc", list)
		case "in-reply-to":
			h.SetMsgIDList("In-Reply-To", vals)
		case "subject":
			subject = strings.Join(vals, ",")
			h.SetText("Subject", subject)
		default:
			// any other header gets ignored on purpose to avoid control headers
			// being injected
		}
	}

	composer, err := NewComposer(nyat, acct, nyat.Config(),
		acct.AccountConfig(), acct.Worker(), "", h, models.OriginalMail{})
	if err != nil {
		return nil
	}
	composer.FocusSubject()
	title := "New email"
	if subject != "" {
		title = subject
		composer.FocusTerminal()
	}
	tab := nyat.NewTab(composer, title)
	composer.OnHeaderChange("Subject", func(subject string) {
		if subject == "" {
			tab.Name = "New email"
		} else {
			tab.Name = subject
		}
		tab.Content.Invalidate()
	})
	return nil
}

func (nyat *Nyat) CloseBackends() error {
	var returnErr error
	for _, acct := range nyat.accounts {
		var raw interface{} = acct.worker.Backend
		c, ok := raw.(io.Closer)
		if !ok {
			continue
		}
		err := c.Close()
		if err != nil {
			returnErr = err
			nyat.logger.Printf("Closing backend failed for %v: %v\n",
				acct.Name(), err)
		}
	}
	return returnErr
}

func (nyat *Nyat) AddDialog(d ui.DrawableInteractive) {
	nyat.dialog = d
	nyat.dialog.OnInvalidate(func(_ ui.Drawable) {
		nyat.Invalidate()
	})
	nyat.Invalidate()
	return
}

func (nyat *Nyat) CloseDialog() {
	nyat.dialog = nil
	nyat.Invalidate()
	return
}

func (nyat *Nyat) GetPassword(title string, prompt string) (chText chan string, chErr chan error) {
	chText = make(chan string, 1)
	chErr = make(chan error, 1)
	getPasswd := NewGetPasswd(title, prompt, nyat.conf, func(pw string, err error) {
		defer func() {
			close(chErr)
			close(chText)
			nyat.CloseDialog()
		}()
		if err != nil {
			chErr <- err
			return
		}
		chErr <- nil
		chText <- pw
		return
	})
	nyat.AddDialog(getPasswd)

	return
}

func (nyat *Nyat) Initialize(ui *ui.UI) {
	nyat.ui = ui
}

func (nyat *Nyat) DecryptKeys(keys []openpgp.Key, symmetric bool) (b []byte, err error) {
	for _, key := range keys {
		ident := key.Entity.PrimaryIdentity()
		chPass, chErr := nyat.GetPassword("Decrypt PGP private key",
			fmt.Sprintf("Enter password for %s (%8X)\nPress <ESC> to cancel",
				ident.Name, key.PublicKey.KeyId))

		for {
			select {
			case err = <-chErr:
				if err != nil {
					return nil, err
				}
				pass := <-chPass
				err = key.PrivateKey.Decrypt([]byte(pass))
				return nil, err
			default:
				nyat.ui.Tick()
			}
		}
	}
	return nil, err
}

// errorScreen is a widget that draws an error in the middle of the context
func errorScreen(s string, conf config.UIConfig) ui.Drawable {
	errstyle := conf.GetStyle(config.STYLE_ERROR)
	text := ui.NewText(s, errstyle).Strategy(ui.TEXT_CENTER)
	grid := ui.NewGrid().Rows([]ui.GridSpec{
		{ui.SIZE_WEIGHT, ui.Const(1)},
		{ui.SIZE_EXACT, ui.Const(1)},
		{ui.SIZE_WEIGHT, ui.Const(1)},
	}).Columns([]ui.GridSpec{
		{ui.SIZE_WEIGHT, ui.Const(1)},
	})
	grid.AddChild(ui.NewFill(' ')).At(0, 0)
	grid.AddChild(text).At(1, 0)
	grid.AddChild(ui.NewFill(' ')).At(2, 0)
	return grid
}

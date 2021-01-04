package config

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"os/exec"
	"path"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode"

	"github.com/gdamore/tcell/v2"
	"github.com/go-ini/ini"
	"github.com/imdario/mergo"
	"github.com/kyoh86/xdg"

	"gitea.com/iwakuramarie/nyat/lib/templates"
)

type GeneralConfig struct {
	DefaultSavePath string `ini:"default-save-path"`
}

type UIConfig struct {
	IndexFormat         string        `ini:"index-format"`
	TimestampFormat     string        `ini:"timestamp-format"`
	ShowHeaders         []string      `delim:","`
	RenderAccountTabs   string        `ini:"render-account-tabs"`
	PinnedTabMarker     string        `ini:"pinned-tab-marker"`
	SidebarWidth        int           `ini:"sidebar-width"`
	PreviewHeight       int           `ini:"preview-height"`
	EmptyMessage        string        `ini:"empty-message"`
	EmptyDirlist        string        `ini:"empty-dirlist"`
	MouseEnabled        bool          `ini:"mouse-enabled"`
	NewMessageBell      bool          `ini:"new-message-bell"`
	Spinner             string        `ini:"spinner"`
	SpinnerDelimiter    string        `ini:"spinner-delimiter"`
	DirListFormat       string        `ini:"dirlist-format"`
	Sort                []string      `delim:" "`
	NextMessageOnDelete bool          `ini:"next-message-on-delete"`
	CompletionDelay     time.Duration `ini:"completion-delay"`
	CompletionPopovers  bool          `ini:"completion-popovers"`
	StyleSetDirs        []string      `ini:"stylesets-dirs" delim:":"`
	StyleSetName        string        `ini:"styleset-name"`
	style               StyleSet
}

type ContextType int

const (
	UI_CONTEXT_FOLDER ContextType = iota
	UI_CONTEXT_ACCOUNT
	UI_CONTEXT_SUBJECT
)

type UIConfigContext struct {
	ContextType ContextType
	Regex       *regexp.Regexp
	UiConfig    UIConfig
}

const (
	FILTER_MIMETYPE = iota
	FILTER_HEADER
)

type AccountConfig struct {
	Archive         string
	CopyTo          string
	Default         string
	Postpone        string
	From            string
	Aliases         string
	Name            string
	Source          string
	SourceCredCmd   string
	Folders         []string
	FoldersExclude  []string
	Params          map[string]string
	Outgoing        string
	OutgoingCredCmd string
	SignatureFile   string
	SignatureCmd    string
	FoldersSort     []string `ini:"folders-sort" delim:","`
}

type BindingConfig struct {
	Global        *KeyBindings
	AccountWizard *KeyBindings
	Compose       *KeyBindings
	ComposeEditor *KeyBindings
	ComposeReview *KeyBindings
	MessageList   *KeyBindings
	MessageView   *KeyBindings
	Terminal      *KeyBindings
}

type ComposeConfig struct {
	Editor         string     `ini:"editor"`
	HeaderLayout   [][]string `ini:"-"`
	AddressBookCmd string     `ini:"address-book-cmd"`
}

type FilterConfig struct {
	FilterType int
	Filter     string
	Command    string
	Header     string
	Regex      *regexp.Regexp
}

type ViewerConfig struct {
	Pager          string
	Alternatives   []string
	ShowHeaders    bool       `ini:"show-headers"`
	AlwaysShowMime bool       `ini:"always-show-mime"`
	HeaderLayout   [][]string `ini:"-"`
}

type TriggersConfig struct {
	NewEmail       string `ini:"new-email"`
	ExecuteCommand func(command []string) error
}

type TemplateConfig struct {
	TemplateDirs []string `ini:"template-dirs", delim:":"`
	QuotedReply  string   `ini:"quoted-reply"`
	Forwards     string   `ini:"forwards"`
}

type NyatConfig struct {
	Bindings      BindingConfig
	Compose       ComposeConfig
	Ini           *ini.File       `ini:"-"`
	Accounts      []AccountConfig `ini:"-"`
	Filters       []FilterConfig  `ini:"-"`
	Viewer        ViewerConfig    `ini:"-"`
	Triggers      TriggersConfig  `ini:"-"`
	Ui            UIConfig
	ContextualUis []UIConfigContext
	General       GeneralConfig
	Templates     TemplateConfig
}

// Input: TimestampFormat
// Output: timestamp-format
func mapName(raw string) string {
	newstr := make([]rune, 0, len(raw))
	for i, chr := range raw {
		if isUpper := 'A' <= chr && chr <= 'Z'; isUpper {
			if i > 0 {
				newstr = append(newstr, '-')
			}
		}
		newstr = append(newstr, unicode.ToLower(chr))
	}
	return string(newstr)
}

func loadAccountConfig(path string) ([]AccountConfig, error) {
	file, err := ini.Load(path)
	if err != nil {
		// No config triggers account configuration wizard
		return nil, nil
	}
	file.NameMapper = mapName

	var accounts []AccountConfig
	for _, _sec := range file.SectionStrings() {
		if _sec == "DEFAULT" {
			continue
		}
		sec := file.Section(_sec)
		account := AccountConfig{
			Archive:  "Archive",
			Default:  "INBOX",
			Postpone: "Drafts",
			Name:     _sec,
			Params:   make(map[string]string),
		}
		if err = sec.MapTo(&account); err != nil {
			return nil, err
		}
		for key, val := range sec.KeysHash() {
			if key == "folders" {
				folders := strings.Split(val, ",")
				sort.Strings(folders)
				account.Folders = folders
			} else if key == "folders-exclude" {
				folders := strings.Split(val, ",")
				sort.Strings(folders)
				account.FoldersExclude = folders
			} else if key == "source-cred-cmd" {
				account.SourceCredCmd = val
			} else if key == "outgoing" {
				account.Outgoing = val
			} else if key == "outgoing-cred-cmd" {
				account.OutgoingCredCmd = val
			} else if key == "from" {
				account.From = val
			} else if key == "aliases" {
				account.Aliases = val
			} else if key == "copy-to" {
				account.CopyTo = val
			} else if key == "archive" {
				account.Archive = val
			} else if key != "name" {
				account.Params[key] = val
			}
		}
		if account.Source == "" {
			return nil, fmt.Errorf("Expected source for account %s", _sec)
		}
		if account.From == "" {
			return nil, fmt.Errorf("Expected from for account %s", _sec)
		}

		source, err := parseCredential(account.Source, account.SourceCredCmd)
		if err != nil {
			return nil, fmt.Errorf("Invalid source credentials for %s: %s", _sec, err)
		}
		account.Source = source

		outgoing, err := parseCredential(account.Outgoing, account.OutgoingCredCmd)
		if err != nil {
			return nil, fmt.Errorf("Invalid outgoing credentials for %s: %s", _sec, err)
		}
		account.Outgoing = outgoing

		accounts = append(accounts, account)
	}
	return accounts, nil
}

func parseCredential(cred, command string) (string, error) {
	if cred == "" || command == "" {
		return cred, nil
	}

	u, err := url.Parse(cred)
	if err != nil {
		return "", err
	}

	// ignore the command if a password is specified
	if _, exists := u.User.Password(); exists {
		return cred, nil
	}

	// don't attempt to parse the command if the url is a path (ie /usr/bin/sendmail)
	if !u.IsAbs() {
		return cred, nil
	}

	cmd := exec.Command("sh", "-c", command)
	cmd.Stdin = os.Stdin
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to read password: %s", err)
	}

	pw := strings.TrimSpace(string(output))
	u.User = url.UserPassword(u.User.Username(), pw)

	return u.String(), nil
}

func installTemplate(root, sharedir, name string) error {
	if _, err := os.Stat(root); os.IsNotExist(err) {
		err := os.MkdirAll(root, 0755)
		if err != nil {
			return err
		}
	}
	data, err := ioutil.ReadFile(path.Join(sharedir, name))
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(path.Join(root, name), data, 0644)
	if err != nil {
		return err
	}
	return nil
}

func (config *NyatConfig) LoadConfig(file *ini.File) error {
	if filters, err := file.GetSection("filters"); err == nil {
		// TODO: Parse the filter more finely, e.g. parse the regex
		for _, match := range filters.KeyStrings() {
			cmd := filters.KeysHash()[match]
			filter := FilterConfig{
				Command: cmd,
				Filter:  match,
			}
			if strings.Contains(match, ",~") {
				filter.FilterType = FILTER_HEADER
				header := filter.Filter[:strings.Index(filter.Filter, ",")]
				regex := filter.Filter[strings.Index(filter.Filter, "~")+1:]
				filter.Header = strings.ToLower(header)
				filter.Regex, err = regexp.Compile(regex)
				if err != nil {
					panic(err)
				}
			} else if strings.ContainsRune(match, ',') {
				filter.FilterType = FILTER_HEADER
				header := filter.Filter[:strings.Index(filter.Filter, ",")]
				value := filter.Filter[strings.Index(filter.Filter, ",")+1:]
				filter.Header = strings.ToLower(header)
				filter.Regex, err = regexp.Compile(regexp.QuoteMeta(value))
			} else {
				filter.FilterType = FILTER_MIMETYPE
			}
			config.Filters = append(config.Filters, filter)
		}
	}
	if viewer, err := file.GetSection("viewer"); err == nil {
		if err := viewer.MapTo(&config.Viewer); err != nil {
			return err
		}
		for key, val := range viewer.KeysHash() {
			switch key {
			case "alternatives":
				config.Viewer.Alternatives = strings.Split(val, ",")
			case "header-layout":
				config.Viewer.HeaderLayout = parseLayout(val)
			}
		}
	}
	if compose, err := file.GetSection("compose"); err == nil {
		if err := compose.MapTo(&config.Compose); err != nil {
			return err
		}
		for key, val := range compose.KeysHash() {
			switch key {
			case "header-layout":
				config.Compose.HeaderLayout = parseLayout(val)
			}
		}
	}
	if ui, err := file.GetSection("ui"); err == nil {
		if err := ui.MapTo(&config.Ui); err != nil {
			return err
		}
	}
	for _, sectionName := range file.SectionStrings() {
		if !strings.Contains(sectionName, "ui:") {
			continue
		}

		uiSection, err := file.GetSection(sectionName)
		if err != nil {
			return err
		}
		uiSubConfig := UIConfig{}
		if err := uiSection.MapTo(&uiSubConfig); err != nil {
			return err
		}
		contextualUi :=
			UIConfigContext{
				UiConfig: uiSubConfig,
			}

		var index int
		if strings.Contains(sectionName, "~") {
			index = strings.Index(sectionName, "~")
			regex := string(sectionName[index+1:])
			contextualUi.Regex, err = regexp.Compile(regex)
			if err != nil {
				return err
			}
		} else if strings.Contains(sectionName, "=") {
			index = strings.Index(sectionName, "=")
			value := string(sectionName[index+1:])
			contextualUi.Regex, err = regexp.Compile(regexp.QuoteMeta(value))
			if err != nil {
				return err
			}
		} else {
			return fmt.Errorf("Invalid Ui Context regex in %s", sectionName)
		}

		switch sectionName[3:index] {
		case "account":
			contextualUi.ContextType = UI_CONTEXT_ACCOUNT
		case "folder":
			contextualUi.ContextType = UI_CONTEXT_FOLDER
		case "subject":
			contextualUi.ContextType = UI_CONTEXT_SUBJECT
		default:
			return fmt.Errorf("Unknown Contextual Ui Section: %s", sectionName)
		}
		config.ContextualUis = append(config.ContextualUis, contextualUi)
	}
	if triggers, err := file.GetSection("triggers"); err == nil {
		if err := triggers.MapTo(&config.Triggers); err != nil {
			return err
		}
	}
	if templatesSec, err := file.GetSection("templates"); err == nil {
		if err := templatesSec.MapTo(&config.Templates); err != nil {
			return err
		}
		templateDirs := templatesSec.Key("template-dirs").String()
		if templateDirs != "" {
			config.Templates.TemplateDirs = strings.Split(templateDirs, ":")
		}
		for key, val := range templatesSec.KeysHash() {
			if key == "template-dirs" {
				continue
			}
			// we want to fail during startup if the templates are not ok
			// hence we do a dummy execute here
			_, err := templates.ParseTemplateFromFile(
				val, config.Templates.TemplateDirs, templates.DummyData())
			if err != nil {
				return err
			}
		}
	}

	if err := config.Ui.loadStyleSet(
		config.Ui.StyleSetDirs); err != nil {
		return err
	}

	for idx, contextualUi := range config.ContextualUis {
		if contextualUi.UiConfig.StyleSetName == "" &&
			len(contextualUi.UiConfig.StyleSetDirs) == 0 {
			continue // no need to do anything if nothing is overridden
		}
		// fill in the missing part from the base
		if contextualUi.UiConfig.StyleSetName == "" {
			config.ContextualUis[idx].UiConfig.StyleSetName = config.Ui.StyleSetName
		} else if len(contextualUi.UiConfig.StyleSetDirs) == 0 {
			config.ContextualUis[idx].UiConfig.StyleSetDirs = config.Ui.StyleSetDirs
		}
		// since at least one of them has changed, load the styleset
		if err := config.ContextualUis[idx].UiConfig.loadStyleSet(
			config.ContextualUis[idx].UiConfig.StyleSetDirs); err != nil {
			return err
		}
	}

	return nil
}

func LoadConfigFromFile(root *string, sharedir string) (*NyatConfig, error) {
	if root == nil {
		_root := path.Join(xdg.ConfigHome(), "nyat")
		root = &_root
	}
	filename := path.Join(*root, "accounts.conf")
	if err := checkConfigPerms(filename); err != nil {
		return nil, err
	}
	filename = path.Join(*root, "nyat.conf")
	file, err := ini.LoadSources(ini.LoadOptions{
		KeyValueDelimiters: "=",
	}, filename)
	if err != nil {
		if err := installTemplate(*root, sharedir, "nyat.conf"); err != nil {
			return nil, err
		}
		if file, err = ini.LoadSources(ini.LoadOptions{
			KeyValueDelimiters: "=",
		}, filename); err != nil {
			return nil, err
		}
	}
	file.NameMapper = mapName
	config := &NyatConfig{
		Bindings: BindingConfig{
			Global:        NewKeyBindings(),
			AccountWizard: NewKeyBindings(),
			Compose:       NewKeyBindings(),
			ComposeEditor: NewKeyBindings(),
			ComposeReview: NewKeyBindings(),
			MessageList:   NewKeyBindings(),
			MessageView:   NewKeyBindings(),
			Terminal:      NewKeyBindings(),
		},
		Ini: file,

		Ui: UIConfig{
			IndexFormat:     "%D %-17.17n %s",
			TimestampFormat: "2006-01-02 03:04 PM",
			ShowHeaders: []string{
				"From", "To", "Cc", "Bcc", "Subject", "Date",
			},
			RenderAccountTabs:   "auto",
			PinnedTabMarker:     "`",
			SidebarWidth:        20,
			PreviewHeight:       12,
			EmptyMessage:        "(no messages)",
			EmptyDirlist:        "(no folders)",
			MouseEnabled:        false,
			NewMessageBell:      true,
			Spinner:             "[..]    , [..]   ,  [..]  ,   [..] ,    [..],   [..] ,  [..]  , [..]   ",
			SpinnerDelimiter:    ",",
			DirListFormat:       "%n %>r",
			NextMessageOnDelete: true,
			CompletionDelay:     250 * time.Millisecond,
			CompletionPopovers:  true,
			StyleSetDirs:        []string{path.Join(sharedir, "stylesets")},
			StyleSetName:        "default",
		},

		ContextualUis: []UIConfigContext{},

		Viewer: ViewerConfig{
			Pager:        "less -R",
			Alternatives: []string{"text/plain", "text/html"},
			ShowHeaders:  false,
			HeaderLayout: [][]string{
				{"From", "To"},
				{"Cc", "Bcc"},
				{"Date"},
				{"Subject"},
			},
		},

		Compose: ComposeConfig{
			HeaderLayout: [][]string{
				{"To", "From"},
				{"Subject"},
			},
		},

		Templates: TemplateConfig{
			TemplateDirs: []string{path.Join(sharedir, "templates")},
			QuotedReply:  "quoted_reply",
			Forwards:     "forward_as_body",
		},
	}

	// These bindings are not configurable
	config.Bindings.AccountWizard.ExKey = KeyStroke{
		Key: tcell.KeyCtrlE,
	}
	quit, _ := ParseBinding("<C-q>", ":quit<Enter>")
	config.Bindings.AccountWizard.Add(quit)

	if err = config.LoadConfig(file); err != nil {
		return nil, err
	}

	if ui, err := file.GetSection("general"); err == nil {
		if err := ui.MapTo(&config.General); err != nil {
			return nil, err
		}
	}

	accountsPath := path.Join(*root, "accounts.conf")
	if accounts, err := loadAccountConfig(accountsPath); err != nil {
		return nil, err
	} else {
		config.Accounts = accounts
	}
	filename = path.Join(*root, "binds.conf")
	binds, err := ini.Load(filename)
	if err != nil {
		if err := installTemplate(*root, sharedir, "binds.conf"); err != nil {
			return nil, err
		}
		if binds, err = ini.Load(filename); err != nil {
			return nil, err
		}
	}
	groups := map[string]**KeyBindings{
		"default":  &config.Bindings.Global,
		"compose":  &config.Bindings.Compose,
		"messages": &config.Bindings.MessageList,
		"terminal": &config.Bindings.Terminal,
		"view":     &config.Bindings.MessageView,

		"compose::editor": &config.Bindings.ComposeEditor,
		"compose::review": &config.Bindings.ComposeReview,
	}
	for _, name := range binds.SectionStrings() {
		sec, err := binds.GetSection(name)
		if err != nil {
			return nil, err
		}
		group, ok := groups[strings.ToLower(name)]
		if !ok {
			return nil, errors.New("Unknown keybinding group " + name)
		}
		bindings := NewKeyBindings()
		for key, value := range sec.KeysHash() {
			if key == "$ex" {
				strokes, err := ParseKeyStrokes(value)
				if err != nil {
					return nil, err
				}
				if len(strokes) != 1 {
					return nil, errors.New(
						"Error: only one keystroke supported for $ex")
				}
				bindings.ExKey = strokes[0]
				continue
			}
			if key == "$noinherit" {
				if value == "false" {
					continue
				}
				if value != "true" {
					return nil, errors.New(
						"Error: expected 'true' or 'false' for $noinherit")
				}
				bindings.Globals = false
				continue
			}
			binding, err := ParseBinding(key, value)
			if err != nil {
				return nil, err
			}
			bindings.Add(binding)
		}
		*group = MergeBindings(bindings, *group)
	}
	// Globals can't inherit from themselves
	config.Bindings.Global.Globals = false
	return config, nil
}

// checkConfigPerms checks for too open permissions
// printing the fix on stdout and returning an error
func checkConfigPerms(filename string) error {
	info, err := os.Stat(filename)
	if err != nil {
		return nil // disregard absent files
	}
	perms := info.Mode().Perm()
	goPerms := perms >> 3
	// group or others have read access
	if goPerms&0x44 != 0 {
		fmt.Fprintf(os.Stderr, "The file %v has too open permissions.\n", filename)
		fmt.Fprintln(os.Stderr, "This is a security issue (it contains passwords).")
		fmt.Fprintf(os.Stderr, "To fix it, run `chmod 600 %v`\n", filename)
		return errors.New("account.conf permissions too lax")
	}
	return nil
}

func parseLayout(layout string) [][]string {
	rows := strings.Split(layout, ",")
	l := make([][]string, len(rows))
	for i, r := range rows {
		l[i] = strings.Split(r, "|")
	}
	return l
}

func (ui *UIConfig) loadStyleSet(styleSetDirs []string) error {
	ui.style = NewStyleSet()
	err := ui.style.LoadStyleSet(ui.StyleSetName, styleSetDirs)
	if err != nil {
		return fmt.Errorf("Unable to load default styleset: %s", err)
	}

	return nil
}

func (config NyatConfig) mergeContextualUi(baseUi UIConfig,
	contextType ContextType, s string) UIConfig {
	for _, contextualUi := range config.ContextualUis {
		if contextualUi.ContextType != contextType {
			continue
		}

		if !contextualUi.Regex.Match([]byte(s)) {
			continue
		}

		mergo.Merge(&baseUi, contextualUi.UiConfig, mergo.WithOverride)
		if contextualUi.UiConfig.StyleSetName != "" {
			baseUi.style = contextualUi.UiConfig.style
		}
		return baseUi
	}

	return baseUi
}

func (config NyatConfig) GetUiConfig(params map[ContextType]string) UIConfig {
	baseUi := config.Ui

	for k, v := range params {
		baseUi = config.mergeContextualUi(baseUi, k, v)
	}

	return baseUi
}

func (uiConfig UIConfig) GetStyle(so StyleObject) tcell.Style {
	return uiConfig.style.Get(so)
}

func (uiConfig UIConfig) GetStyleSelected(so StyleObject) tcell.Style {
	return uiConfig.style.Selected(so)
}

func (uiConfig UIConfig) GetComposedStyle(base StyleObject,
	styles []StyleObject) tcell.Style {
	return uiConfig.style.Compose(base, styles)
}

func (uiConfig UIConfig) GetComposedStyleSelected(base StyleObject, styles []StyleObject) tcell.Style {
	return uiConfig.style.ComposeSelected(base, styles)
}

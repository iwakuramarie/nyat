package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"sort"
	"time"

	"git.sr.ht/~sircmpwn/getopt"
	"github.com/mattn/go-isatty"

	"gitea.com/iwakuramarie/nyat/commands"
	"gitea.com/iwakuramarie/nyat/commands/account"
	"gitea.com/iwakuramarie/nyat/commands/compose"
	"gitea.com/iwakuramarie/nyat/commands/msg"
	"gitea.com/iwakuramarie/nyat/commands/msgview"
	"gitea.com/iwakuramarie/nyat/commands/terminal"
	"gitea.com/iwakuramarie/nyat/config"
	"gitea.com/iwakuramarie/nyat/lib"
	"gitea.com/iwakuramarie/nyat/lib/templates"
	libui "gitea.com/iwakuramarie/nyat/lib/ui"
	"gitea.com/iwakuramarie/nyat/widgets"
)

func getCommands(selected libui.Drawable) []*commands.Commands {
	switch selected.(type) {
	case *widgets.AccountView:
		return []*commands.Commands{
			account.AccountCommands,
			msg.MessageCommands,
			commands.GlobalCommands,
		}
	case *widgets.Composer:
		return []*commands.Commands{
			compose.ComposeCommands,
			commands.GlobalCommands,
		}
	case *widgets.MessageViewer:
		return []*commands.Commands{
			msgview.MessageViewCommands,
			msg.MessageCommands,
			commands.GlobalCommands,
		}
	case *widgets.Terminal:
		return []*commands.Commands{
			terminal.TerminalCommands,
			commands.GlobalCommands,
		}
	default:
		return []*commands.Commands{commands.GlobalCommands}
	}
}

func execCommand(nyat *widgets.Nyat, ui *libui.UI, cmd []string) error {
	cmds := getCommands((*nyat).SelectedTab())
	for i, set := range cmds {
		err := set.ExecuteCommand(nyat, cmd)
		if _, ok := err.(commands.NoSuchCommand); ok {
			if i == len(cmds)-1 {
				return err
			}
			continue
		} else if _, ok := err.(commands.ErrorExit); ok {
			ui.Exit()
			return nil
		} else if err != nil {
			return err
		} else {
			break
		}
	}
	return nil
}

func getCompletions(nyat *widgets.Nyat, cmd string) []string {
	var completions []string
	for _, set := range getCommands((*nyat).SelectedTab()) {
		completions = append(completions, set.GetCompletions(nyat, cmd)...)
	}
	sort.Strings(completions)
	return completions
}

var (
	ShareDir string
	Version  string
)

func usage() {
	log.Fatal("Usage: nyat [-v] [mailto:...]")
}

func main() {
	opts, optind, err := getopt.Getopts(os.Args, "v")
	if err != nil {
		log.Print(err)
		usage()
		return
	}
	for _, opt := range opts {
		switch opt.Option {
		case 'v':
			fmt.Println("nyat " + Version)
			return
		}
	}
	initDone := make(chan struct{})
	args := os.Args[optind:]
	if len(args) > 1 {
		usage()
		return
	} else if len(args) == 1 {
		arg := args[0]
		err := lib.ConnectAndExec(arg)
		if err == nil {
			return // other nyat instance takes over
		}
		fmt.Fprintf(os.Stderr, "Failed to communicate to nyat: %v", err)
		// continue with setting up a new nyat instance and retry after init
		go func(msg string) {
			<-initDone
			err := lib.ConnectAndExec(msg)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Failed to communicate to nyat: %v", err)
			}
		}(arg)
	}

	var (
		logOut io.Writer
		logger *log.Logger
	)
	if !isatty.IsTerminal(os.Stdout.Fd()) {
		logOut = os.Stdout
	} else {
		logOut = ioutil.Discard
		os.Stdout, _ = os.Open(os.DevNull)
	}
	logger = log.New(logOut, "", log.LstdFlags)
	logger.Println("Starting up nyat")

	conf, err := config.LoadConfigFromFile(nil, ShareDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load config: %v\n", err)
		os.Exit(1)
	}

	var (
		nyat *widgets.Nyat
		ui   *libui.UI
	)

	nyat = widgets.NewNyat(conf, logger, func(cmd []string) error {
		return execCommand(nyat, ui, cmd)
	}, func(cmd string) []string {
		return getCompletions(nyat, cmd)
	}, &commands.CmdHistory)

	ui, err = libui.Initialize(nyat)
	if err != nil {
		panic(err)
	}
	defer ui.Close()

	if conf.Ui.MouseEnabled {
		ui.EnableMouse()
	}

	logger.Println("Initializing PGP keyring")
	lib.InitKeyring()
	defer lib.UnlockKeyring()

	logger.Println("Starting Unix server")
	as, err := lib.StartServer(logger)
	if err != nil {
		logger.Printf("Failed to start Unix server: %v (non-fatal)", err)
	} else {
		defer as.Close()
		as.OnMailto = nyat.Mailto
	}

	// set the nyat version so that we can use it in the template funcs
	templates.SetVersion(Version)

	close(initDone)

	for !ui.ShouldExit() {
		for nyat.Tick() {
			// Continue updating our internal state
		}
		if !ui.Tick() {
			// ~60 FPS
			time.Sleep(16 * time.Millisecond)
		}
	}
	nyat.CloseBackends()
}

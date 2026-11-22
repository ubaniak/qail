package cmd

import (
	"context"
	"log"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/ubaniak/qail/internal/config"
	"github.com/ubaniak/qail/internal/console"
	"github.com/ubaniak/qail/internal/qailhome"
)

var (
	consolePort   int
	consoleNoOpen bool
)

var consoleCmd = &cobra.Command{
	Use:   "console [workspace] [repo]",
	Short: "Run the Agent Console and open it in the browser",
	Long: `Start the Agent Console — a local web UI for Claude Code agents scoped
to qail workspaces and repos — and open it in the default browser. Pass a
workspace (and optionally a repo) to open straight into that scope.

The console is served by this qail process on 127.0.0.1 and runs agents with
your installed Claude Code CLI (` + "`claude`" + `; set QAIL_CLAUDE to pick another).
If the desktop app already serves the console, this just opens it. See
ADR-0020.`,
	Args: cobra.MaximumNArgs(2),
	Run: func(_ *cobra.Command, args []string) {
		m := newConsole(mustStore(), consolePort)

		ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
		defer stop()

		if err := m.Start(ctx); err != nil {
			log.Fatal(err)
		}
		var ws, repo string
		if len(args) > 0 {
			ws = args[0]
		}
		if len(args) > 1 {
			repo = args[1]
		}
		url := m.URL(console.ScopePath(ws, repo))
		if !consoleNoOpen {
			if err := m.OpenBrowser(ctx, url); err != nil {
				log.Printf("%v", err)
			}
		}

		// Someone else (e.g. the menubar app) already serves the console:
		// nothing for this process to keep running.
		if !m.Running() {
			log.Printf("agent console already running at %s", url)
			return
		}
		if !m.ClaudeFound() {
			log.Printf("warning: %v", console.ErrNoClaude)
		}
		log.Printf("agent console at %s (Ctrl-C to stop)", url)
		<-ctx.Done()
		m.Stop()
	},
}

// newConsole builds the Agent Console for this process. Chat threads live
// in ~/.qail/console/data, where earlier (Node) releases kept them too.
func newConsole(s config.Store, port int) *console.Manager {
	return console.New(console.Options{
		Store:   s,
		DataDir: filepath.Join(qailHome().ConsoleDir(), "data"),
		Port:    port,
		Log:     log.Writer(),
	})
}

func qailHome() qailhome.Home {
	h, err := qailhome.Default()
	if err != nil {
		log.Fatalf("qail home: %v", err)
	}
	return h
}

func init() {
	consoleCmd.Flags().IntVar(&consolePort, "port", console.DefaultPort, "console port (loopback only)")
	consoleCmd.Flags().BoolVar(&consoleNoOpen, "no-open", false, "don't open the browser")
	rootCmd.AddCommand(consoleCmd)
}

package cmd

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/ubaniak/qail/internal/console"
	"github.com/ubaniak/qail/internal/qailhome"
)

var (
	consolePort    int
	consoleQailURL string
	consoleNoOpen  bool
)

var consoleCmd = &cobra.Command{
	Use:   "console [workspace] [repo]",
	Short: "Run the Agent Console and open it in the browser",
	Long: `Start the Agent Console — a local web UI for Claude Code agents scoped
to qail workspaces and repos — and open it in the default browser. Pass a
workspace (and optionally a repo) to open straight into that scope.

The console reads workspaces from the qail HTTP API; if nothing is serving
--qail-url it starts ` + "`qail serve`" + ` itself. Install the console once with
` + "`make console-install`" + ` (see ADR-0018).`,
	Args: cobra.MaximumNArgs(2),
	Run: func(_ *cobra.Command, args []string) {
		self, _ := os.Executable()
		m := console.New(console.Options{
			Dir:           consoleDir(),
			Port:          consolePort,
			QailURL:       consoleQailURL,
			QailBin:       self,
			AutostartQail: true,
			Log:           os.Stderr,
		})

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

		// Someone else (e.g. the menubar app) already runs the console:
		// nothing for this process to supervise.
		if !m.Running() {
			log.Printf("agent console already running at %s", url)
			return
		}
		log.Printf("agent console at %s (Ctrl-C to stop)", url)
		<-ctx.Done()
		m.Stop()
	},
}

// consoleDir resolves the console install: $QAIL_CONSOLE_DIR, else
// qailhome's ConsoleDir.
func consoleDir() string {
	if d := os.Getenv("QAIL_CONSOLE_DIR"); d != "" {
		return d
	}
	h, err := qailhome.Default()
	if err != nil {
		log.Fatalf("qail home: %v", err)
	}
	return h.ConsoleDir()
}

func init() {
	consoleCmd.Flags().IntVar(&consolePort, "port", console.DefaultPort, "console port (loopback only)")
	consoleCmd.Flags().StringVar(&consoleQailURL, "qail-url", "http://127.0.0.1:8765", "qail HTTP API the console reads workspaces from")
	consoleCmd.Flags().BoolVar(&consoleNoOpen, "no-open", false, "don't open the browser")
	rootCmd.AddCommand(consoleCmd)
}

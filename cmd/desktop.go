package cmd

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"

	qailapp "github.com/ubaniak/qail/internal/app"
	"github.com/ubaniak/qail/internal/console"
)

var desktopBackendCmd = &cobra.Command{
	Use:    "desktop-backend",
	Short:  "Serve the desktop app's RPC bridge over stdin/stdout",
	Hidden: true,
	Long: `Internal: spawned by the Electron desktop app. Reads newline-delimited
JSON requests on stdin and writes responses and progress events to stdout.
Not meant to be run by hand.`,
	Run: func(cmd *cobra.Command, _ []string) {
		// stdout is the protocol channel. Anything else that would print
		// there (an action logging progress, a subprocess inheriting
		// os.Stdout) is redirected to stderr so it can't corrupt a frame.
		proto := os.Stdout
		os.Stdout = os.Stderr
		log.SetOutput(os.Stderr)

		ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
		defer stop()

		store := mustStore()

		// Agent Console (ADR-0020): served in-process on demand. The tray
		// drives it through Bindings; it stops when Electron closes our stdin.
		agentConsole := newConsole(store, console.DefaultPort)

		srv := qailapp.NewServer(ctx, store, proto, qailapp.WithConsole(agentConsole))
		err := srv.Serve(os.Stdin)
		agentConsole.Stop()
		if err != nil {
			log.Fatalf("desktop-backend: %v", err)
		}
	},
}

func init() {
	rootCmd.AddCommand(desktopBackendCmd)
}

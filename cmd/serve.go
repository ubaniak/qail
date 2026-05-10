package cmd

import (
	"context"
	"errors"
	"log"
	"net"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/ubaniak/qail/internal/api"
)

// flag-backed inputs for `qail serve`. Defaults aim at "local web UI on
// the same machine as the CLI": loopback only, common dev port.
var (
	serveAddr string
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Run the qail HTTP API for the web UI",
	Long: `Serve qail's actions over HTTP so a web UI can drive the same
operations as the CLI. Default bind is 127.0.0.1:8765 (loopback only) — no
authentication is performed, so do not expose this on a public interface
without a reverse proxy that adds auth.`,
	Run: func(cmd *cobra.Command, _ []string) {
		s := mustStore()
		srv := api.New(s)

		httpSrv := &http.Server{
			Addr:              serveAddr,
			Handler:           srv.Handler(),
			ReadHeaderTimeout: 10 * time.Second,
			// No write timeout: SSE streams (workspace create) can run
			// arbitrarily long; the per-request context already gates
			// cancellation when the client disconnects.
		}

		ln, err := net.Listen("tcp", serveAddr)
		if err != nil {
			log.Fatalf("listen %s: %v", serveAddr, err)
		}
		log.Printf("qail api listening on %s", ln.Addr())

		// Graceful shutdown on SIGINT/SIGTERM. Without this, an in-flight
		// SSE stream gets a hard reset on Ctrl-C; with it, we close the
		// listener and let active handlers drain for a few seconds.
		ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
		defer stop()

		go func() {
			if err := httpSrv.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
				log.Printf("server error: %v", err)
				stop()
			}
		}()

		<-ctx.Done()
		log.Println("shutting down")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := httpSrv.Shutdown(shutdownCtx); err != nil {
			log.Printf("graceful shutdown failed: %v", err)
		}
	},
}

func init() {
	serveCmd.Flags().StringVar(&serveAddr, "addr", "127.0.0.1:8765", "TCP address to bind (host:port)")
	rootCmd.AddCommand(serveCmd)
}

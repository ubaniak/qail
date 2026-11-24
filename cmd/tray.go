package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/ncruces/zenity"
	"github.com/spf13/cobra"

	qailapp "github.com/ubaniak/qail/internal/app"
	"github.com/ubaniak/qail/internal/console"
	"github.com/ubaniak/qail/internal/tray"
	"github.com/ubaniak/qail/internal/ui"
)

// defaultTrayPort sits next to the Agent Console's port (4317).
const defaultTrayPort = 4318

var (
	trayPort int
	trayOpen bool
)

var trayCmd = &cobra.Command{
	Use:   "tray",
	Short: "Run qail in the system tray, with its UI in your browser",
	Long: `Put a qail icon in the system tray (the menu bar on macOS). Left-click
opens the qail UI in your default browser; right-click shows a menu with the
Agent Console and Quit.

The UI is served by this process on 127.0.0.1 only (default port 4318), so
other machines can't reach it. Running ` + "`qail tray`" + ` again while one is
already running just opens the UI. See ADR-0021.`,
	Args: cobra.NoArgs,
	Run: func(_ *cobra.Command, _ []string) {
		detachConsole()

		uiURL := "http://127.0.0.1:" + strconv.Itoa(trayPort) + "/"
		ln, err := net.Listen("tcp", net.JoinHostPort("127.0.0.1", strconv.Itoa(trayPort)))
		if err != nil {
			// Another qail tray already owns the port: show its UI, the
			// way a second launch of the Electron app shows its window.
			if trayAlreadyRunning(uiURL) {
				if err := openURL(uiURL); err != nil {
					log.Fatal(err)
				}
				return
			}
			log.Fatalf("qail tray: %v", err)
		}

		ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
		defer stop()

		store := mustStore()
		agentConsole := newConsole(store, console.DefaultPort)
		srv := qailapp.NewHTTPServer(ctx, store, ui.FS(), pickDirectory, qailapp.WithConsole(agentConsole))
		httpSrv := &http.Server{
			Handler:           srv.Handler(),
			ReadHeaderTimeout: 10 * time.Second,
			// No write timeout: /api/events streams for as long as a tab is open.
		}
		go func() {
			if err := httpSrv.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
				log.Printf("qail tray: %v", err)
				tray.Quit()
			}
		}()
		log.Printf("qail UI at %s", uiURL)

		// Ctrl-C in a terminal quits the tray like its Quit item does.
		go func() {
			<-ctx.Done()
			tray.Quit()
		}()

		tray.Run(tray.Options{
			UIURL:   uiURL,
			OpenURL: openURL,
			Console: srv.Bindings(),
			Log:     log.Writer(),
			OnReady: func() {
				if trayOpen {
					if err := openURL(uiURL); err != nil {
						log.Print(err)
					}
				}
			},
			OnExit: func() {
				stop()
				shutdownCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
				defer cancel()
				_ = httpSrv.Shutdown(shutdownCtx)
				agentConsole.Stop()
			},
		})
	},
}

// trayAlreadyRunning reports whether a qail tray answers at base.
func trayAlreadyRunning(base string) bool {
	c := &http.Client{Timeout: time.Second}
	res, err := c.Get(base + "api/health")
	if err != nil {
		return false
	}
	defer res.Body.Close()
	var body struct {
		App string `json:"app"`
	}
	return json.NewDecoder(res.Body).Decode(&body) == nil && body.App == qailapp.HTTPAppName
}

// openURL opens u in the default browser.
func openURL(u string) error {
	return console.New(console.Options{Log: log.Writer()}).OpenBrowser(context.Background(), u)
}

// pickDirectory backs the UI's Browse button with the OS folder dialog.
// On Linux zenity needs the zenity, qarma or matedialog program; without
// one the UI says so and the path can still be typed.
func pickDirectory(ctx context.Context, title string) (string, error) {
	opts := []zenity.Option{zenity.Directory(), zenity.Context(ctx)}
	if title != "" {
		opts = append(opts, zenity.Title(title))
	}
	if home, err := os.UserHomeDir(); err == nil {
		opts = append(opts, zenity.Filename(home+string(os.PathSeparator)))
	}
	path, err := zenity.SelectFile(opts...)
	if errors.Is(err, zenity.ErrCanceled) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("folder picker: %w", err)
	}
	return path, nil
}

func init() {
	trayCmd.Flags().IntVar(&trayPort, "port", defaultTrayPort, "UI port (loopback only)")
	trayCmd.Flags().BoolVar(&trayOpen, "open", false, "open the UI in the browser on start")
	rootCmd.AddCommand(trayCmd)
}

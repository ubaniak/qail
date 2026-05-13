package cmd

import (
	"embed"
	"io/fs"
	"log"

	"github.com/spf13/cobra"
	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"

	qailapp "github.com/ubaniak/qail/internal/app"
)

// frontendAssets holds the embedded Vite bundle injected by
// assets_app.go (build-tag-gated) via SetFrontendAssets. Plain
// `go build` keeps it nil, in which case `qail app` fails loud rather
// than launching an empty window.
var frontendAssets fs.FS

// trayIcon is the PNG bytes used for the systray icon. Set via
// SetTrayIcon at init time from an embed in internal/app/.
var trayIcon []byte

// SetFrontendAssets is called from assets_app.go before Execute. The
// argument is the root embed.FS containing `frontend/dist/...`; we
// re-root onto dist so Wails serves index.html as `/`.
func SetFrontendAssets(efs embed.FS) {
	sub, err := fs.Sub(efs, "frontend/dist")
	if err != nil {
		log.Fatalf("frontend assets: %v", err)
	}
	frontendAssets = sub
}

// SetTrayIcon stashes the bytes of the menubar icon. Wired from
// internal/app/icon.go (build-tagged the same as the frontend).
func SetTrayIcon(b []byte) { trayIcon = b }

var appCmd = &cobra.Command{
	Use:   "app",
	Short: "Launch the qail menubar app",
	Long: `Run qail as a menubar (system tray) app. Click the tray icon to
toggle a small frameless window anchored beneath it. The same actions
package is shared with the CLI and HTTP server, so changes made in the
app appear immediately to qail commands in any terminal.`,
	Run: func(cmd *cobra.Command, _ []string) {
		if frontendAssets == nil {
			log.Fatal("frontend assets not embedded; rebuild via `make app`")
		}

		// ActivationPolicyAccessory == no Dock icon, no Cmd-Tab entry.
		// ApplicationShouldTerminateAfterLastWindowClosed=false lets the
		// window be hidden without quitting the process — exactly what
		// a menubar app wants.
		app := application.New(application.Options{
			Name:        "qail",
			Description: "Multi-repo workspace manager",
			Assets: application.AssetOptions{
				Handler: application.AssetFileServerFS(frontendAssets),
			},
			Mac: application.MacOptions{
				ActivationPolicy: application.ActivationPolicyAccessory,
				ApplicationShouldTerminateAfterLastWindowClosed: false,
			},
		})

		bindings := qailapp.New(mustStore(), app)
		app.RegisterService(application.NewServiceWithOptions(
			bindings,
			application.ServiceOptions{Name: "Bindings"},
		))

		// Window: 480x640 frameless dark panel. Start hidden — clicking
		// the tray reveals it.
		win := app.Window.NewWithOptions(application.WebviewWindowOptions{
			Title:     "qail",
			Width:     480,
			Height:    640,
			MinWidth:  400,
			MinHeight: 500,
			Frameless: true,
			Mac: application.MacWindow{
				InvisibleTitleBarHeight: 0,
				Backdrop:                application.MacBackdropTranslucent,
				TitleBar:                application.MacTitleBarHiddenInset,
			},
			BackgroundColour: application.NewRGB(9, 9, 11), // zinc-950
			URL:              "/",
		})

		// Closing the window only hides it; the tray click reopens it.
		// Without Cancel() the window would be destroyed and re-show
		// would 404 the asset server.
		win.RegisterHook(events.Common.WindowClosing, func(e *application.WindowEvent) {
			win.Hide()
			e.Cancel()
		})
		win.Hide()

		// Drag region wiring — matches the data-drag CSS in
		// frontend/src/index.css.
		// (In v3 the drag attribute is handled by webview-side CSS,
		// no separate option needed.)

		tray := app.SystemTray.New()
		if len(trayIcon) > 0 {
			tray.SetIcon(trayIcon)
		} else {
			tray.SetLabel("Q")
		}
		// AttachWindow makes left-click toggle the window beneath the
		// tray icon; right-click still shows the menu below.
		tray.AttachWindow(win).WindowOffset(5)

		menu := app.NewMenu()
		menu.Add("Show qail").OnClick(func(*application.Context) { win.Show() })
		menu.AddSeparator()
		menu.Add("Quit").OnClick(func(*application.Context) { app.Quit() })
		tray.SetMenu(menu)

		// Global hotkey ⌃⌥Q toggles the tray window. Carbon fires the
		// channel on the main run loop; the goroutine just dispatches to
		// the existing show/hide path. Registration failure is logged but
		// non-fatal so the app still launches without the shortcut.
		if ch, err := qailapp.RegisterToggleHotkey(); err != nil {
			log.Printf("global hotkey disabled: %v", err)
		} else {
			go func() {
				for range ch {
					if win.IsVisible() {
						win.Hide()
					} else {
						win.Show()
						win.Focus()
					}
				}
			}()
		}

		if err := app.Run(); err != nil {
			log.Fatal(err)
		}
	},
}

func init() {
	rootCmd.AddCommand(appCmd)
}

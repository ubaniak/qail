//go:build production

// Package main embeds the built frontend + the menubar icon when the
// binary is compiled with `-tags production` (the tag set by
// `wails3 build` and `make app`). Plain `go build .` skips this file so
// the CLI builds without requiring a vite output directory to exist;
// `qail app` then returns a clear error if launched from a non-prod
// binary.
package main

import (
	"embed"

	"github.com/ubaniak/qail/cmd"
)

//go:embed all:frontend/dist
var frontendDist embed.FS

//go:embed icon.png
var trayIcon []byte

func init() {
	cmd.SetFrontendAssets(frontendDist)
	cmd.SetTrayIcon(trayIcon)
}

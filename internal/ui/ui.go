// Package ui embeds the built React frontend for `qail tray` (ADR-0021).
//
// `make frontend` builds frontend/ and copies frontend/dist here; a plain
// `make build` embeds only the placeholder, and the tray then serves a page
// saying the UI wasn't built. The directory is git-ignored apart from
// .gitkeep, which keeps the embed pattern valid on a fresh checkout.
package ui

import (
	"embed"
	"io/fs"
)

//go:embed all:dist
var dist embed.FS

// FS returns the frontend build, with index.html at its root when built.
func FS() fs.FS {
	sub, err := fs.Sub(dist, "dist")
	if err != nil {
		panic(err) // the embed pattern guarantees dist exists
	}
	return sub
}

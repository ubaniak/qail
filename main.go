package main

import (
	"os"
	"strings"

	"github.com/ubaniak/qail/cmd"
)

func main() {
	// When launched from a macOS .app bundle (Finder double-click, Dock,
	// Spotlight, `open qail.app`), launchd execs the binary at
	// `.../qail.app/Contents/MacOS/qail` with no arguments. Cobra would
	// then print --help to a non-existent stdout and exit silently. Treat
	// that case as `qail app`.
	if len(os.Args) == 1 && strings.Contains(os.Args[0], ".app/Contents/MacOS/") {
		os.Args = append(os.Args, "app")
	}
	cmd.Execute()
}

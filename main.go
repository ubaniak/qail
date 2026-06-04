package main

import (
	"os"
	"strings"

	"github.com/ubaniak/qail/cmd"
)

func main() {
	// macOS GUI launches (Finder, Dock, Spotlight, `open qail.app`) inherit
	// the launchd PATH (`/usr/bin:/bin:/usr/sbin:/sbin`), so Homebrew bins
	// like tmux/git in /usr/local/bin or /opt/homebrew/bin are invisible to
	// exec.LookPath. Prepend the common Homebrew prefixes before any cmd
	// runs.
	ensurePath("/opt/homebrew/bin", "/usr/local/bin")

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

func ensurePath(dirs ...string) {
	path := os.Getenv("PATH")
	parts := strings.Split(path, ":")
	seen := make(map[string]bool, len(parts))
	for _, p := range parts {
		seen[p] = true
	}
	var prepend []string
	for _, d := range dirs {
		if !seen[d] {
			prepend = append(prepend, d)
			seen[d] = true
		}
	}
	if len(prepend) == 0 {
		return
	}
	os.Setenv("PATH", strings.Join(prepend, ":")+":"+path)
}

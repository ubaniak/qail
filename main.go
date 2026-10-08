package main

import (
	"os"
	"strings"

	"github.com/ubaniak/qail/cmd"
)

func main() {
	// macOS GUI launches (Finder, Dock, Spotlight, and the Electron app
	// spawning `qail desktop-backend`) inherit
	// the launchd PATH (`/usr/bin:/bin:/usr/sbin:/sbin`), so Homebrew bins
	// like tmux/git in /usr/local/bin or /opt/homebrew/bin are invisible to
	// exec.LookPath. Prepend the common Homebrew prefixes before any cmd
	// runs.
	ensurePath("/opt/homebrew/bin", "/usr/local/bin")

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

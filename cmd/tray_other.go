//go:build !windows

package cmd

// detachConsole is a no-op off Windows; see tray_windows.go.
func detachConsole() {}

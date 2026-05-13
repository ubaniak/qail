//go:build !darwin

package app

// RegisterToggleHotkey is a stub on non-darwin builds. The menubar app
// only ships on macOS today; the channel is returned so callers don't
// need OS-specific branching at the call site.
func RegisterToggleHotkey() (<-chan struct{}, error) {
	return make(chan struct{}), nil
}

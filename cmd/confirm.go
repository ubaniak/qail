package cmd

import "github.com/ubaniak/qail/internal/forms"

// confirmOrSkip enforces the cmd-layer confirmation contract for destructive
// non-interactive ops. Three outcomes:
//
//   - --yes set: returns (true, nil) without prompting.
//   - --no-tty set without --yes: returns the requireTTY error so the
//     caller fails fast with a clear "pass --yes" message.
//   - otherwise: opens forms.Confirm(promptMsg) and returns its result.
//
// Centralising this means future policy tweaks (e.g. an "are you really
// sure?" double-confirm flag) live in one place instead of being smeared
// across every remove handler.
func confirmOrSkip(promptMsg, ttyReason string) (bool, error) {
	if flagYes {
		return true, nil
	}
	if err := requireTTY(ttyReason); err != nil {
		return false, err
	}
	return forms.Confirm(promptMsg)
}

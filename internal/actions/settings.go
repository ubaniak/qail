package actions

import "github.com/ubaniak/qail/internal/config"

// SetRoot updates the qail Root path. An empty value is allowed because the
// existing `qail config root` flow accepts whatever the user types.
func SetRoot(s config.Store, root string) error {
	return readWrite(s, func(cfg *config.Config) error {
		cfg.Root = root
		return nil
	})
}

// SetEditor updates the editor binary used by `qail workspace open`.
func SetEditor(s config.Store, editor string) error {
	return readWrite(s, func(cfg *config.Config) error {
		cfg.Editor = editor
		return nil
	})
}

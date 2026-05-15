package actions

import (
	"fmt"

	"github.com/ubaniak/qail/internal/config"
)

// SetRoot updates the qail Root path. An empty value is allowed because the
// existing `qail config root` flow accepts whatever the user types.
func SetRoot(s config.Store, root string) error {
	return readWrite(s, func(cfg *config.Config) error {
		cfg.Root = root
		return nil
	})
}

// AddEditor registers a named editor. Name is the identity key; command is
// the executable invoked by `qail workspace open`. Returns an error on
// duplicate name or empty inputs.
func AddEditor(s config.Store, name, command string) error {
	if name == "" {
		return fmt.Errorf("editor name must not be empty")
	}
	if command == "" {
		return fmt.Errorf("editor command must not be empty")
	}
	return readWrite(s, func(cfg *config.Config) error {
		if _, ok := findEditor(*cfg, name); ok {
			return fmt.Errorf("editor %q already exists", name)
		}
		cfg.Editors = append(cfg.Editors, config.Editor{Name: name, Command: command})
		if cfg.DefaultEditor == "" {
			cfg.DefaultEditor = name
		}
		return nil
	})
}

// RemoveEditor deletes the named editor and clears any refs pointing at it
// (workspace overrides + global default).
func RemoveEditor(s config.Store, name string) error {
	if name == "" {
		return fmt.Errorf("editor name must not be empty")
	}
	return readWrite(s, func(cfg *config.Config) error {
		idx := -1
		for i, e := range cfg.Editors {
			if e.Name == name {
				idx = i
				break
			}
		}
		if idx == -1 {
			return fmt.Errorf("editor %q not found", name)
		}
		cfg.Editors = append(cfg.Editors[:idx], cfg.Editors[idx+1:]...)
		if cfg.DefaultEditor == name {
			cfg.DefaultEditor = ""
		}
		for wsName, profile := range cfg.Workspaces {
			if profile.Editor == name {
				profile.Editor = ""
				cfg.Workspaces[wsName] = profile
			}
		}
		return nil
	})
}

// SetDefaultEditor sets the global default editor. Name must reference an
// existing editor.
func SetDefaultEditor(s config.Store, name string) error {
	return readWrite(s, func(cfg *config.Config) error {
		if name != "" {
			if _, ok := findEditor(*cfg, name); !ok {
				return fmt.Errorf("editor %q not found", name)
			}
		}
		cfg.DefaultEditor = name
		return nil
	})
}

// SetWorkspaceEditor overrides the editor for a workspace. Empty name
// clears the override (workspace inherits global default).
func SetWorkspaceEditor(s config.Store, workspace, name string) error {
	if workspace == "" {
		return fmt.Errorf("workspace must not be empty")
	}
	return readWrite(s, func(cfg *config.Config) error {
		profile, ok := cfg.Workspaces[workspace]
		if !ok {
			return fmt.Errorf("workspace %q not found", workspace)
		}
		if name != "" {
			if _, ok := findEditor(*cfg, name); !ok {
				return fmt.Errorf("editor %q not found", name)
			}
		}
		profile.Editor = name
		cfg.Workspaces[workspace] = profile
		return nil
	})
}

// ListEditors returns the registered editors plus the global default name.
func ListEditors(s config.Store) ([]config.Editor, string, error) {
	cfg, err := s.Read()
	if err != nil {
		return nil, "", err
	}
	return cfg.Editors, cfg.DefaultEditor, nil
}

// findEditor returns the editor with the given name and ok=true if found.
func findEditor(cfg config.Config, name string) (config.Editor, bool) {
	for _, e := range cfg.Editors {
		if e.Name == name {
			return e, true
		}
	}
	return config.Editor{}, false
}

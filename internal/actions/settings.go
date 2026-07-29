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

// AddAI registers a named AI tool. Name is the identity key; command is
// the executable invoked by `qail workspace ai-open`. Returns an error on
// duplicate name or empty inputs.
func AddAI(s config.Store, name, command string) error {
	if name == "" {
		return fmt.Errorf("ai name must not be empty")
	}
	if command == "" {
		return fmt.Errorf("ai command must not be empty")
	}
	return readWrite(s, func(cfg *config.Config) error {
		if _, ok := findAI(*cfg, name); ok {
			return fmt.Errorf("ai %q already exists", name)
		}
		cfg.AIs = append(cfg.AIs, config.AI{Name: name, Command: command})
		if cfg.DefaultAI == "" {
			cfg.DefaultAI = name
		}
		return nil
	})
}

// RemoveAI deletes the named AI tool and clears any refs pointing at it
// (workspace overrides + global default).
func RemoveAI(s config.Store, name string) error {
	if name == "" {
		return fmt.Errorf("ai name must not be empty")
	}
	return readWrite(s, func(cfg *config.Config) error {
		idx := -1
		for i, a := range cfg.AIs {
			if a.Name == name {
				idx = i
				break
			}
		}
		if idx == -1 {
			return fmt.Errorf("ai %q not found", name)
		}
		cfg.AIs = append(cfg.AIs[:idx], cfg.AIs[idx+1:]...)
		if cfg.DefaultAI == name {
			cfg.DefaultAI = ""
		}
		for wsName, profile := range cfg.Workspaces {
			if profile.AI == name {
				profile.AI = ""
				cfg.Workspaces[wsName] = profile
			}
		}
		return nil
	})
}

// SetDefaultAI sets the global default AI tool. Name must reference an
// existing AI tool.
func SetDefaultAI(s config.Store, name string) error {
	return readWrite(s, func(cfg *config.Config) error {
		if name != "" {
			if _, ok := findAI(*cfg, name); !ok {
				return fmt.Errorf("ai %q not found", name)
			}
		}
		cfg.DefaultAI = name
		return nil
	})
}

// SetWorkspaceAI overrides the AI tool for a workspace. Empty name clears
// the override (workspace inherits global default).
func SetWorkspaceAI(s config.Store, workspace, name string) error {
	if workspace == "" {
		return fmt.Errorf("workspace must not be empty")
	}
	return readWrite(s, func(cfg *config.Config) error {
		profile, ok := cfg.Workspaces[workspace]
		if !ok {
			return fmt.Errorf("workspace %q not found", workspace)
		}
		if name != "" {
			if _, ok := findAI(*cfg, name); !ok {
				return fmt.Errorf("ai %q not found", name)
			}
		}
		profile.AI = name
		cfg.Workspaces[workspace] = profile
		return nil
	})
}

// ListAI returns the registered AI tools plus the global default name.
func ListAI(s config.Store) ([]config.AI, string, error) {
	cfg, err := s.Read()
	if err != nil {
		return nil, "", err
	}
	return cfg.AIs, cfg.DefaultAI, nil
}

// findAI returns the AI tool with the given name and ok=true if found.
func findAI(cfg config.Config, name string) (config.AI, bool) {
	for _, a := range cfg.AIs {
		if a.Name == name {
			return a, true
		}
	}
	return config.AI{}, false
}

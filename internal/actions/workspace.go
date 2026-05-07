package actions

import (
	"fmt"
	"time"

	"github.com/ubaniak/qail/internal/config"
	"github.com/ubaniak/qail/internal/workspace"
)

// AddWorkspace creates a new workspace on disk (cloning each package and
// running post-install scripts) and registers it in the config. If disk
// creation fails the registry is not updated, matching the old
// HandleConfig behaviour.
func AddWorkspace(s config.Store, name string, packages []string) error {
	if name == "" {
		return fmt.Errorf("workspace name must not be empty")
	}
	cfg, err := s.Read()
	if err != nil {
		return err
	}
	if err := buildAndCreate(cfg, name, packages); err != nil {
		return err
	}
	if cfg.Workspaces == nil {
		cfg.Workspaces = make(config.Workspace)
	}
	cfg.Workspaces[name] = config.NewWorkspaceProfile(packages, time.Now().UTC())
	return s.Write(cfg)
}

// EditWorkspace updates an existing workspace's package list, rebuilds it
// on disk, and persists the change. Returns an error if the workspace is
// not registered.
func EditWorkspace(s config.Store, name string, packages []string) error {
	cfg, err := s.Read()
	if err != nil {
		return err
	}
	if _, ok := cfg.Workspaces[name]; !ok {
		return fmt.Errorf("workspace %q not found", name)
	}
	if err := buildAndCreate(cfg, name, packages); err != nil {
		return err
	}
	cfg.Workspaces[name] = config.NewWorkspaceProfile(packages, time.Now().UTC())
	return s.Write(cfg)
}

// CloneWorkspace registers dstName with the given packages and creates it
// on disk. The caller is responsible for resolving the source workspace's
// package list (typically via forms.CloneWorkspace).
func CloneWorkspace(s config.Store, dstName string, packages []string) error {
	return AddWorkspace(s, dstName, packages)
}

// CreateWorkspaceOnDisk (re)materialises an already-registered workspace
// directory without touching the registry. Used by `qail ws create` when
// the user wants to rebuild a workspace from scratch.
func CreateWorkspaceOnDisk(s config.Store, name string) error {
	cfg, err := s.Read()
	if err != nil {
		return err
	}
	profile, ok := cfg.Workspaces[name]
	if !ok {
		return fmt.Errorf("workspace %q not found", name)
	}
	return buildAndCreate(cfg, name, profile.Repos)
}

// RemoveWorkspace deletes a workspace from the registry and strips the
// workspace from the post-install-script attachments. The on-disk
// directory is not removed; `qail ws clean` handles orphans.
func RemoveWorkspace(s config.Store, name string) error {
	if name == "" {
		return nil
	}
	return readWrite(s, func(cfg *config.Config) error {
		delete(cfg.Workspaces, name)
		delete(cfg.PostInstallScripts.Workspace, name)
		return nil
	})
}

// TouchWorkspace updates LastUsed for the named workspace and returns the
// previous profile so callers can resolve packages and paths. Used by
// open / cd / mux — every flow that bumps LastUsed plus performs a side
// effect (open editor, copy command).
func TouchWorkspace(s config.Store, name string) (config.WorkspaceProfile, error) {
	var profile config.WorkspaceProfile
	err := readWrite(s, func(cfg *config.Config) error {
		p, ok := cfg.Workspaces[name]
		if !ok {
			return fmt.Errorf("workspace %q not found", name)
		}
		profile = p
		cfg.Workspaces[name] = config.NewWorkspaceProfile(p.Repos, time.Now().UTC())
		return nil
	})
	return profile, err
}

// ListWorkspaces returns the workspace registry and the per-workspace
// post-install-script attachments. Read-only; matches ListRepos.
func ListWorkspaces(s config.Store) (config.Workspace, map[string][]string, error) {
	cfg, err := s.Read()
	if err != nil {
		return nil, nil, err
	}
	return cfg.Workspaces, cfg.PostInstallScripts.Workspace, nil
}

// SetWorkspacePostInstall replaces the post-install script list for a
// workspace. An empty scripts slice clears the entry.
func SetWorkspacePostInstall(s config.Store, name string, scripts []string) error {
	if name == "" {
		return fmt.Errorf("workspace must not be empty")
	}
	return readWrite(s, func(cfg *config.Config) error {
		if cfg.PostInstallScripts.Workspace == nil {
			cfg.PostInstallScripts.Workspace = make(map[string][]string)
		}
		if len(scripts) == 0 {
			delete(cfg.PostInstallScripts.Workspace, name)
			return nil
		}
		cfg.PostInstallScripts.Workspace[name] = append([]string(nil), scripts...)
		return nil
	})
}

// WorkspaceContext is the read-only slice of config used by flows that
// don't mutate (clean, list, explore). Returning a value type keeps cmd
// handlers from needing to import config.Store directly.
type WorkspaceContext struct {
	Root       string
	Editor     string
	Workspaces config.Workspace
}

// ReadWorkspaceContext returns the cfg fields needed by read-only flows.
func ReadWorkspaceContext(s config.Store) (WorkspaceContext, error) {
	cfg, err := s.Read()
	if err != nil {
		return WorkspaceContext{}, err
	}
	return WorkspaceContext{
		Root:       cfg.Root,
		Editor:     cfg.Editor,
		Workspaces: cfg.Workspaces,
	}, nil
}

// buildAndCreate is the shared "build a Workspace and Create() it"
// helper. Lives here so AddWorkspace, EditWorkspace, and
// CreateWorkspaceOnDisk all wire packages and post-install scripts the
// same way.
func buildAndCreate(cfg config.Config, name string, packages []string) error {
	ws := workspace.NewDefault(cfg.Root, name, packages, cfg.Repos)
	ws.WithRepoPostInstallScripts(cfg.PostInstallScripts.Repo)
	ws.WithWSPostInstallScripts(cfg.PostInstallScripts.Workspace)
	return ws.Create()
}

package actions

import (
	"context"
	"fmt"
	"io"
	"os"
	"path"
	"time"

	"github.com/ubaniak/qail/internal/config"
	"github.com/ubaniak/qail/internal/installer"
	"github.com/ubaniak/qail/internal/scripts"
	"github.com/ubaniak/qail/internal/workspace"
)

// osStat is overridable in tests; production uses os.Stat. Restore
// validates orphan presence by Stat'ing the resolved path.
var osStat = os.Stat

// AddWorkspace creates a new workspace on disk (cloning each package and
// running post-install scripts) and registers it in the config. The
// postInstall slice attaches workspace-scoped scripts to the new
// workspace before the build runs, so they fire as part of creation.
// If disk creation fails the registry is not updated. Progress is
// written to out (nil → os.Stdout).
func AddWorkspace(ctx context.Context, s config.Store, name string, packages []string, postInstall []string, out io.Writer) error {
	if name == "" {
		return fmt.Errorf("workspace name must not be empty")
	}
	if err := validateScopedScripts(scripts.ScopeWorkspace, postInstall); err != nil {
		return err
	}
	storeMu.Lock()
	cfg, err := s.Read()
	if err == nil && len(postInstall) > 0 {
		if cfg.PostInstallScripts.Workspace == nil {
			cfg.PostInstallScripts.Workspace = make(map[string][]string)
		}
		cfg.PostInstallScripts.Workspace[name] = append([]string(nil), postInstall...)
	}
	storeMu.Unlock()
	if err != nil {
		return err
	}
	if err := buildAndCreate(ctx, cfg, name, packages, out); err != nil {
		return err
	}
	return readWrite(s, func(c *config.Config) error {
		if c.Workspaces == nil {
			c.Workspaces = make(config.Workspace)
		}
		c.Workspaces[name] = config.NewWorkspaceProfile(packages, time.Now().UTC())
		if len(postInstall) > 0 {
			if c.PostInstallScripts.Workspace == nil {
				c.PostInstallScripts.Workspace = make(map[string][]string)
			}
			c.PostInstallScripts.Workspace[name] = append([]string(nil), postInstall...)
		}
		return nil
	})
}

// EditWorkspace updates an existing workspace's package list, rebuilds it
// on disk, and persists the change. Returns an error if the workspace is
// not registered.
func EditWorkspace(ctx context.Context, s config.Store, name string, packages []string, out io.Writer) error {
	storeMu.Lock()
	cfg, err := s.Read()
	storeMu.Unlock()
	if err != nil {
		return err
	}
	if _, ok := cfg.Workspaces[name]; !ok {
		return fmt.Errorf("workspace %q not found", name)
	}
	if err := buildAndCreate(ctx, cfg, name, packages, out); err != nil {
		return err
	}
	return readWrite(s, func(c *config.Config) error {
		if _, ok := c.Workspaces[name]; !ok {
			return fmt.Errorf("workspace %q not found", name)
		}
		c.Workspaces[name] = config.NewWorkspaceProfile(packages, time.Now().UTC())
		return nil
	})
}

// CloneWorkspace registers dstName with the given packages and creates it
// on disk. The caller is responsible for resolving the source workspace's
// package list (typically via forms.CloneWorkspace). The cloned workspace
// starts with no post-install scripts; callers can attach them afterwards.
func CloneWorkspace(ctx context.Context, s config.Store, dstName string, packages []string, out io.Writer) error {
	return AddWorkspace(ctx, s, dstName, packages, nil, out)
}

// CreateWorkspaceOnDisk (re)materialises an already-registered workspace
// directory without touching the registry. Used by `qail ws create` when
// the user wants to rebuild a workspace from scratch.
func CreateWorkspaceOnDisk(ctx context.Context, s config.Store, name string, out io.Writer) error {
	storeMu.Lock()
	cfg, err := s.Read()
	storeMu.Unlock()
	if err != nil {
		return err
	}
	profile, ok := cfg.Workspaces[name]
	if !ok {
		return fmt.Errorf("workspace %q not found", name)
	}
	return buildAndCreate(ctx, cfg, name, profile.Repos, out)
}

// RemoveWorkspace deletes a workspace from the registry but leaves its
// post-install-script attachments in place so a later RestoreWorkspace
// can re-bind them. The on-disk directory is not removed either; `qail
// ws clean` is the "kill it for good" path and drops the script entries.
func RemoveWorkspace(s config.Store, name string) error {
	if name == "" {
		return nil
	}
	return readWrite(s, func(cfg *config.Config) error {
		delete(cfg.Workspaces, name)
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
// workspace. An empty scripts slice clears the entry. Each name must
// exist in the workspace scripts bucket; cross-scope references are
// rejected so the post-install runner can never resolve to a repo script.
func SetWorkspacePostInstall(s config.Store, name string, scriptList []string) error {
	if name == "" {
		return fmt.Errorf("workspace must not be empty")
	}
	if err := validateScopedScripts(scripts.ScopeWorkspace, scriptList); err != nil {
		return err
	}
	return readWrite(s, func(cfg *config.Config) error {
		if cfg.PostInstallScripts.Workspace == nil {
			cfg.PostInstallScripts.Workspace = make(map[string][]string)
		}
		if len(scriptList) == 0 {
			delete(cfg.PostInstallScripts.Workspace, name)
			return nil
		}
		cfg.PostInstallScripts.Workspace[name] = append([]string(nil), scriptList...)
		return nil
	})
}

// RunWorkspaceScript executes a workspace-scoped script against the
// named workspace's on-disk path. Progress streams to out. The script
// does not need to be currently attached to the workspace; this is the
// "run on demand" entrypoint.
func RunWorkspaceScript(ctx context.Context, s config.Store, workspaceName, scriptName string, out io.Writer) error {
	if workspaceName == "" {
		return fmt.Errorf("workspace must not be empty")
	}
	if scriptName == "" {
		return fmt.Errorf("script must not be empty")
	}
	cfg, err := s.Read()
	if err != nil {
		return err
	}
	if _, ok := cfg.Workspaces[workspaceName]; !ok {
		return fmt.Errorf("workspace %q not found", workspaceName)
	}
	dir := path.Join(cfg.Root, workspaceName)
	inst := installer.NewStreaming()
	return inst.RunPostInstall(ctx, scripts.ScopeWorkspace, dir, []string{scriptName}, out)
}

// validateScopedScripts returns an error if any name in list is missing
// from the given scope's bucket. Empty list is a no-op (clears the
// attachment list).
func validateScopedScripts(scope scripts.Scope, list []string) error {
	if len(list) == 0 {
		return nil
	}
	sc := scripts.Default()
	for _, name := range list {
		ok, err := sc.Has(scope, name)
		if err != nil {
			return err
		}
		if !ok {
			return fmt.Errorf("script %q not found in %s scope", name, scope)
		}
	}
	return nil
}

// WorkspaceContext is the read-only slice of config used by flows that
// don't mutate (clean, list, explore). Returning a value type keeps cmd
// handlers from needing to import config.Store directly.
type WorkspaceContext struct {
	Root       string
	Workspaces config.Workspace
}

// ListOrphanWorkspaces lists directories under cfg.Root that are not
// registered as workspaces — candidates for cleanup. Reads config once;
// the caller may pair it with RemoveOrphanWorkspaces after confirmation.
func ListOrphanWorkspaces(s config.Store) ([]string, error) {
	cfg, err := s.Read()
	if err != nil {
		return nil, err
	}
	if cfg.Root == "" {
		return nil, nil
	}
	return workspace.ListOrphans(cfg.Root, cfg.Workspaces)
}

// OrphanPath returns the absolute on-disk path for an orphan directory
// under cfg.Root. Refuses if name is empty, root unconfigured, or name is
// actually a registered workspace (those use the workspace flow). Pure;
// no editor spawn — clipboard/open callers compose with it.
func OrphanPath(s config.Store, name string) (string, error) {
	if name == "" {
		return "", fmt.Errorf("orphan name must not be empty")
	}
	cfg, err := s.Read()
	if err != nil {
		return "", err
	}
	if cfg.Root == "" {
		return "", fmt.Errorf("workspace root is not configured")
	}
	if _, tracked := cfg.Workspaces[name]; tracked {
		return "", fmt.Errorf("%q is a registered workspace; use the workspace flow", name)
	}
	return path.Join(cfg.Root, name), nil
}

// RemoveOrphanWorkspaces deletes the named subdirectories of cfg.Root
// and forgets any post-install-script attachments preserved for restore.
// Defensive: refuses to touch any name that is currently a registered
// workspace, even if the caller asked — registered workspaces have their
// own remove flow that also strips config entries.
func RemoveOrphanWorkspaces(s config.Store, names []string) error {
	storeMu.Lock()
	cfg, err := s.Read()
	if err != nil {
		storeMu.Unlock()
		return err
	}
	if cfg.Root == "" {
		storeMu.Unlock()
		return fmt.Errorf("workspace root is not configured")
	}
	for _, name := range names {
		if _, tracked := cfg.Workspaces[name]; tracked {
			storeMu.Unlock()
			return fmt.Errorf("%q is a registered workspace; remove it via the workspace list, not cleanup", name)
		}
	}
	if err := workspace.RemoveOrphans(cfg.Root, names); err != nil {
		storeMu.Unlock()
		return err
	}
	for _, name := range names {
		delete(cfg.PostInstallScripts.Workspace, name)
	}
	err = s.Write(cfg)
	storeMu.Unlock()
	return err
}

// OrphanInspection is the data the restore UI needs to render the
// confirm modal: orphan path, detected repos (with match info), and the
// workspace-scoped post-install scripts still attached from before the
// remove. Scripts attached to repos survive automatically since they're
// keyed by repo name, not workspace name — they aren't surfaced here.
type OrphanInspection struct {
	Path             string
	Repos            []workspace.DetectedRepo
	PreservedScripts []string
}

// InspectOrphan resolves the orphan path, walks its subdirs with go-git,
// and returns DetectedRepo entries plus the preserved workspace scripts
// from the registry. Refuses if name is empty, root unconfigured, or
// name is actually a registered workspace.
func InspectOrphan(s config.Store, name string) (OrphanInspection, error) {
	if name == "" {
		return OrphanInspection{}, fmt.Errorf("orphan name must not be empty")
	}
	cfg, err := s.Read()
	if err != nil {
		return OrphanInspection{}, err
	}
	if cfg.Root == "" {
		return OrphanInspection{}, fmt.Errorf("workspace root is not configured")
	}
	if _, tracked := cfg.Workspaces[name]; tracked {
		return OrphanInspection{}, fmt.Errorf("%q is a registered workspace; use the workspace flow", name)
	}
	wsPath := path.Join(cfg.Root, name)
	repos, err := workspace.DetectRepos(wsPath, cfg.Repos)
	if err != nil {
		return OrphanInspection{}, err
	}
	preserved := append([]string(nil), cfg.PostInstallScripts.Workspace[name]...)
	return OrphanInspection{Path: wsPath, Repos: repos, PreservedScripts: preserved}, nil
}

// RestoreWorkspace re-registers an orphan directory as a workspace
// attached to the given repo names. Repos must already exist in
// cfg.Repos; unknown names are rejected so the registry stays
// internally consistent. The on-disk dir must exist (the user is
// restoring it, not creating from scratch). Workspace-scoped scripts
// preserved through the prior remove are kept in place, so the restored
// workspace inherits its old post-install attachments automatically.
func RestoreWorkspace(s config.Store, name string, repos []string) error {
	if name == "" {
		return fmt.Errorf("workspace name must not be empty")
	}
	return readWrite(s, func(cfg *config.Config) error {
		if cfg.Root == "" {
			return fmt.Errorf("workspace root is not configured")
		}
		if _, exists := cfg.Workspaces[name]; exists {
			return fmt.Errorf("workspace %q is already registered", name)
		}
		wsPath := path.Join(cfg.Root, name)
		info, err := osStat(wsPath)
		if err != nil {
			return fmt.Errorf("orphan directory not found: %v", err)
		}
		if !info.IsDir() {
			return fmt.Errorf("%s is not a directory", wsPath)
		}
		for _, r := range repos {
			if _, ok := cfg.Repos[r]; !ok {
				return fmt.Errorf("repo %q not registered", r)
			}
		}
		if cfg.Workspaces == nil {
			cfg.Workspaces = make(config.Workspace)
		}
		cfg.Workspaces[name] = config.NewWorkspaceProfile(append([]string(nil), repos...), time.Now().UTC())
		return nil
	})
}

// ReadWorkspaceContext returns the cfg fields needed by read-only flows.
func ReadWorkspaceContext(s config.Store) (WorkspaceContext, error) {
	cfg, err := s.Read()
	if err != nil {
		return WorkspaceContext{}, err
	}
	return WorkspaceContext{
		Root:       cfg.Root,
		Workspaces: cfg.Workspaces,
	}, nil
}

// buildAndCreate is the shared "build a Workspace and Create() it"
// helper. Lives here so AddWorkspace, EditWorkspace, and
// CreateWorkspaceOnDisk all wire packages and post-install scripts the
// same way. When out is non-nil it goes through workspace.NewStreaming
// (no huh spinner) so HTTP/SSE callers see plain progress lines; when
// out is nil the spinner-driven default is used.
func buildAndCreate(ctx context.Context, cfg config.Config, name string, packages []string, out io.Writer) error {
	var ws workspace.Workspace
	if out == nil {
		ws = workspace.NewDefault(cfg.Root, name, packages, cfg.Repos)
	} else {
		ws = workspace.NewStreaming(cfg.Root, name, packages, cfg.Repos)
	}
	ws.WithRepoPostInstallScripts(cfg.PostInstallScripts.Repo)
	ws.WithWSPostInstallScripts(cfg.PostInstallScripts.Workspace)
	return ws.Create(ctx, out)
}

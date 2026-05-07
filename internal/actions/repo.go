package actions

import (
	"fmt"

	"github.com/ubaniak/qail/internal/config"
)

// AddRepo registers a repo (Name -> URL) in the config. If the name already
// exists its URL is overwritten — the caller decides whether that's an
// update or a mistake worth confirming.
func AddRepo(s config.Store, name, url string) error {
	if name == "" {
		return fmt.Errorf("repo name must not be empty")
	}
	return readWrite(s, func(cfg *config.Config) error {
		if cfg.Repos == nil {
			cfg.Repos = make(map[string]string)
		}
		cfg.Repos[name] = url
		return nil
	})
}

// RemoveRepos deletes every name in names from cfg.Repos and also strips
// the same names from every workspace's repo list and from the per-repo
// post-install scripts map. The cascade lives inside the action so callers
// cannot leave dangling references.
func RemoveRepos(s config.Store, names []string) error {
	if len(names) == 0 {
		return nil
	}
	set := make(map[string]struct{}, len(names))
	for _, n := range names {
		set[n] = struct{}{}
	}
	return readWrite(s, func(cfg *config.Config) error {
		for n := range set {
			delete(cfg.Repos, n)
			delete(cfg.PostInstallScripts.Repo, n)
		}
		for wsName, ws := range cfg.Workspaces {
			filtered := ws.Repos[:0:0]
			for _, r := range ws.Repos {
				if _, drop := set[r]; !drop {
					filtered = append(filtered, r)
				}
			}
			ws.Repos = filtered
			cfg.Workspaces[wsName] = ws
		}
		return nil
	})
}

// SetRepoPostInstall replaces the post-install script list for the given
// repo. An empty scripts slice clears the entry.
func SetRepoPostInstall(s config.Store, repo string, scripts []string) error {
	if repo == "" {
		return fmt.Errorf("repo must not be empty")
	}
	return readWrite(s, func(cfg *config.Config) error {
		if cfg.PostInstallScripts.Repo == nil {
			cfg.PostInstallScripts.Repo = make(map[string][]string)
		}
		if len(scripts) == 0 {
			delete(cfg.PostInstallScripts.Repo, repo)
			return nil
		}
		cfg.PostInstallScripts.Repo[repo] = append([]string(nil), scripts...)
		return nil
	})
}

// ListRepos returns the repo URL map and the per-repo post-install scripts
// map. It is read-only; the function exists so cmd handlers do not import
// config.Store directly for trivial reads.
func ListRepos(s config.Store) (map[string]string, map[string][]string, error) {
	cfg, err := s.Read()
	if err != nil {
		return nil, nil, err
	}
	return cfg.Repos, cfg.PostInstallScripts.Repo, nil
}

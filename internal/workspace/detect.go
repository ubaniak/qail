package workspace

import (
	"os"
	"path/filepath"

	"github.com/go-git/go-git/v5"
)

// DetectedRepo describes one subdirectory of a workspace path. RemoteURL
// is the origin URL when the dir is a git repo with an "origin" remote;
// "" if the dir is not a repo or has no origin. Match is the key in the
// caller-supplied known map that the dir maps to (URL match wins; dir
// name fallback); "" when nothing matches.
type DetectedRepo struct {
	Dir       string
	RemoteURL string
	Match     string
}

// DetectRepos inspects every immediate subdirectory of wsPath using
// go-git and matches each against the known repos map (name -> URL).
// Files and dotfiles are ignored. Bare/no-git/no-origin dirs are
// returned with empty RemoteURL so the UI can show them as "unknown"
// alongside matched repos.
func DetectRepos(wsPath string, known map[string]string) ([]DetectedRepo, error) {
	entries, err := os.ReadDir(wsPath)
	if err != nil {
		return nil, err
	}

	urlToName := make(map[string]string, len(known))
	for name, url := range known {
		if url != "" {
			urlToName[url] = name
		}
	}

	var out []DetectedRepo
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		name := e.Name()
		if len(name) > 0 && name[0] == '.' {
			continue
		}
		dr := DetectedRepo{Dir: name}
		repo, err := git.PlainOpen(filepath.Join(wsPath, name))
		if err == nil {
			if remote, err := repo.Remote("origin"); err == nil {
				if urls := remote.Config().URLs; len(urls) > 0 {
					dr.RemoteURL = urls[0]
				}
			}
		}
		if dr.RemoteURL != "" {
			if match, ok := urlToName[dr.RemoteURL]; ok {
				dr.Match = match
			}
		}
		if dr.Match == "" {
			if _, ok := known[name]; ok {
				dr.Match = name
			}
		}
		out = append(out, dr)
	}
	return out, nil
}

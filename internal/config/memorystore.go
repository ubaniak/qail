package config

import "sync"

// MemoryStore is the in-process Store adapter, used by tests and any caller
// that wants Config persistence without a SQLite file. Read returns a deep
// copy of the held Config so callers can mutate freely without affecting
// the store; Write replaces the held Config with a deep copy of the input.
type MemoryStore struct {
	mu  sync.Mutex
	cfg Config
}

// NewMemoryStore returns an empty MemoryStore. Initial Read returns a
// zero-value Config with empty PostInstallScripts maps.
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{cfg: zeroConfig()}
}

// NewMemoryStoreFrom seeds a MemoryStore with cfg.
func NewMemoryStoreFrom(cfg Config) *MemoryStore {
	return &MemoryStore{cfg: cloneConfig(cfg)}
}

func (m *MemoryStore) Read() (Config, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return cloneConfig(m.cfg), nil
}

func (m *MemoryStore) Write(cfg Config) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.cfg = cloneConfig(cfg)
	return nil
}

// zeroConfig returns a Config with the post-install map fields initialised
// (matching the convention SQLiteStore.Read uses on first Read).
func zeroConfig() Config {
	return Config{
		PostInstallScripts: PostInstallScripts{
			Repo:      make(map[string][]string),
			Workspace: make(map[string][]string),
		},
	}
}

// cloneConfig returns a deep copy. The Workspace map and the post-install
// maps are non-trivial; without cloning, callers and the store would alias
// the same slice/map memory.
func cloneConfig(cfg Config) Config {
	out := Config{
		Root:          cfg.Root,
		DefaultEditor: cfg.DefaultEditor,
		DefaultAI:     cfg.DefaultAI,
	}
	if cfg.Editors != nil {
		out.Editors = append([]Editor(nil), cfg.Editors...)
	}
	if cfg.AIs != nil {
		out.AIs = append([]AI(nil), cfg.AIs...)
	}
	if cfg.Repos != nil {
		out.Repos = make(map[string]string, len(cfg.Repos))
		for k, v := range cfg.Repos {
			out.Repos[k] = v
		}
	}
	if cfg.Workspaces != nil {
		out.Workspaces = make(Workspace, len(cfg.Workspaces))
		for k, v := range cfg.Workspaces {
			repos := append([]string(nil), v.Repos...)
			out.Workspaces[k] = WorkspaceProfile{
				Repos:    repos,
				LastUsed: v.LastUsed,
				Editor:   v.Editor,
				AI:       v.AI,
			}
		}
	}
	out.PostInstallScripts = PostInstallScripts{
		Repo:      cloneStringSliceMap(cfg.PostInstallScripts.Repo),
		Workspace: cloneStringSliceMap(cfg.PostInstallScripts.Workspace),
	}
	return out
}

func cloneStringSliceMap(m map[string][]string) map[string][]string {
	if m == nil {
		return make(map[string][]string)
	}
	out := make(map[string][]string, len(m))
	for k, v := range m {
		out[k] = append([]string(nil), v...)
	}
	return out
}

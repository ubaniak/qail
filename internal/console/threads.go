package console

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"sync"
)

// Chat threads, one per (scope, agent), persisted as JSON in
// DataDir/store.json. The format is the one the Node console wrote, so
// threads from earlier releases carry over.

// Thread is one agent's conversation in one scope.
type Thread struct {
	SessionID     string        `json:"sessionId"`
	Messages      []ChatMessage `json:"messages"`
	SkillsUsed    []string      `json:"skillsUsed"`
	RuntimeSkills []string      `json:"runtimeSkills,omitempty"`
	Updated       string        `json:"updated"`
}

// ChatMessage is one user or assistant message as the UI renders it.
type ChatMessage struct {
	Role   string    `json:"role"`
	Text   string    `json:"text"`
	Ts     string    `json:"ts"`
	Tools  []ToolRef `json:"tools,omitempty"`
	Skills []string  `json:"skills,omitempty"`
	Cost   *float64  `json:"cost,omitempty"`
}

// ToolRef is a tool call shown under an assistant message. Input is a
// truncated JSON string.
type ToolRef struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Sub   bool   `json:"sub"`
	Input string `json:"input"`
}

type threadStore struct {
	path string

	saveMu  sync.Mutex // one writer of store.json at a time
	mu      sync.Mutex
	threads map[string]*Thread
}

func threadKey(scope, agent string) string { return scope + "::" + agent }

func openThreadStore(dir string) (*threadStore, error) {
	ts := &threadStore{path: filepath.Join(dir, "store.json"), threads: map[string]*Thread{}}
	if dir == "" {
		ts.path = ""
		return ts, nil
	}
	b, err := os.ReadFile(ts.path)
	if errors.Is(err, fs.ErrNotExist) {
		return ts, nil
	}
	if err != nil {
		return nil, err
	}
	var disk struct {
		Threads map[string]*Thread `json:"threads"`
	}
	if err := json.Unmarshal(b, &disk); err != nil {
		return nil, err
	}
	for k, t := range disk.Threads {
		if t != nil {
			ts.threads[k] = t
		}
	}
	return ts, nil
}

// with runs fn under the store lock. Callers that change anything save
// afterwards.
func (ts *threadStore) with(fn func(map[string]*Thread)) {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	fn(ts.threads)
}

// get returns a copy of the thread, or nil.
func (ts *threadStore) get(key string) *Thread {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	t := ts.threads[key]
	if t == nil {
		return nil
	}
	c := *t
	c.Messages = append([]ChatMessage(nil), t.Messages...)
	c.SkillsUsed = append([]string(nil), t.SkillsUsed...)
	c.RuntimeSkills = append([]string(nil), t.RuntimeSkills...)
	return &c
}

func (ts *threadStore) delete(key string) error {
	ts.mu.Lock()
	delete(ts.threads, key)
	ts.mu.Unlock()
	return ts.save()
}

// save writes the store atomically (temp file + rename).
func (ts *threadStore) save() error {
	if ts.path == "" {
		return nil
	}
	ts.saveMu.Lock()
	defer ts.saveMu.Unlock()
	ts.mu.Lock()
	b, err := json.MarshalIndent(map[string]any{"threads": ts.threads}, "", "  ")
	ts.mu.Unlock()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(ts.path), 0o755); err != nil {
		return err
	}
	tmp := ts.path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, ts.path)
}

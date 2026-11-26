package console

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// Discovery of the agents, skills and /shipit runs visible in a scope,
// read the way Claude Code reads them: ~/.claude/{agents,skills} for the
// user, <scope>/.claude/{agents,skills} for the project (project wins).

const mainAgent = "main"

// Skill is a SKILL.md under a skills directory.
type Skill struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Scope       string   `json:"scope"`
	File        string   `json:"file"`
	Error       string   `json:"error,omitempty"`
	PreloadedBy []string `json:"preloadedBy"`
}

// Agent is a subagent definition (or the default "main" session).
type Agent struct {
	Name               string       `json:"name"`
	Scope              string       `json:"scope"`
	Description        string       `json:"description"`
	Model              string       `json:"model"`
	Tools              []string     `json:"tools"`
	PreloadedSkills    []string     `json:"preloadedSkills"`
	MissingSkills      []string     `json:"missingSkills"`
	CanInvokeSkills    bool         `json:"canInvokeSkills"`
	DiscoverableSkills []string     `json:"discoverableSkills"`
	Prompt             string       `json:"prompt"`
	File               string       `json:"file,omitempty"`
	Error              string       `json:"error,omitempty"`
	Thread             *threadBrief `json:"thread"`
	Busy               bool         `json:"busy"`
}

type threadBrief struct {
	Count      int      `json:"count"`
	SkillsUsed []string `json:"skillsUsed"`
	Updated    string   `json:"updated"`
}

// ShipRun is a /shipit run directory under <scope>/.agent/runs.
type ShipRun struct {
	ID      string         `json:"id"`
	State   map[string]any `json:"state"`
	Files   []string       `json:"files"`
	Updated string         `json:"updated"`
}

type mdFile struct {
	data    map[string]any
	content string
	file    string
	slug    string
	err     string
}

// parseFrontmatter splits a leading `---` YAML block from the body.
func parseFrontmatter(b []byte) (map[string]any, string, error) {
	b = bytes.TrimPrefix(b, []byte("\xef\xbb\xbf"))
	s := strings.ReplaceAll(string(b), "\r\n", "\n")
	if !strings.HasPrefix(s, "---\n") {
		return map[string]any{}, s, nil
	}
	rest := s[4:]
	end := strings.Index(rest, "\n---")
	if end < 0 {
		return map[string]any{}, s, nil
	}
	head, body := rest[:end], rest[end+4:]
	if i := strings.IndexByte(body, '\n'); i >= 0 {
		body = body[i+1:]
	} else {
		body = ""
	}
	data := map[string]any{}
	if err := yaml.Unmarshal([]byte(head), &data); err != nil {
		return map[string]any{}, body, err
	}
	if data == nil {
		data = map[string]any{}
	}
	return data, body, nil
}

// readMdDir reads dir/*.md ("agents") or dir/*/SKILL.md ("skills").
func readMdDir(dir, kind string) []mdFile {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var out []mdFile
	for _, e := range entries {
		var file, slug string
		switch {
		case kind == "agents" && !e.IsDir() && strings.HasSuffix(e.Name(), ".md"):
			file, slug = filepath.Join(dir, e.Name()), strings.TrimSuffix(e.Name(), ".md")
		case kind == "skills" && isDir(filepath.Join(dir, e.Name())):
			file, slug = filepath.Join(dir, e.Name(), "SKILL.md"), e.Name()
		default:
			continue
		}
		b, err := os.ReadFile(file)
		if err != nil {
			continue
		}
		data, content, err := parseFrontmatter(b)
		f := mdFile{data: data, content: content, file: file, slug: slug}
		if err != nil {
			f.err = err.Error()
		}
		out = append(out, f)
	}
	return out
}

func isDir(p string) bool {
	st, err := os.Stat(p) // follows symlinked skill dirs
	return err == nil && st.IsDir()
}

func str(v any) string {
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return fmt.Sprint(v)
}

// splitList accepts a YAML list or a comma-separated string.
func splitList(v any) []string {
	switch t := v.(type) {
	case []any:
		out := make([]string, 0, len(t))
		for _, x := range t {
			out = append(out, str(x))
		}
		return out
	case string:
		var out []string
		for _, p := range strings.Split(t, ",") {
			if p = strings.TrimSpace(p); p != "" {
				out = append(out, p)
			}
		}
		return out
	}
	return nil
}

func discoverSkills(homeClaude, projectPath string) []Skill {
	byName := map[string]Skill{}
	for _, src := range [][2]string{{"user", filepath.Join(homeClaude, "skills")}, {"project", filepath.Join(projectPath, ".claude", "skills")}} {
		for _, f := range readMdDir(src[1], "skills") {
			name := str(f.data["name"])
			if name == "" {
				name = f.slug
			}
			byName[name] = Skill{Name: name, Description: str(f.data["description"]), Scope: src[0], File: f.file, Error: f.err, PreloadedBy: []string{}}
		}
	}
	out := make([]Skill, 0, len(byName))
	for _, s := range byName {
		out = append(out, s)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

func discoverAgents(homeClaude, projectPath string, skills []Skill) []Agent {
	skillNames := make([]string, 0, len(skills))
	have := map[string]bool{}
	for _, s := range skills {
		skillNames = append(skillNames, s.Name)
		have[s.Name] = true
	}
	byName := map[string]Agent{}
	for _, src := range [][2]string{{"user", filepath.Join(homeClaude, "agents")}, {"project", filepath.Join(projectPath, ".claude", "agents")}} {
		for _, f := range readMdDir(src[1], "agents") {
			name := str(f.data["name"])
			if name == "" {
				name = f.slug
			}
			tools := splitList(f.data["tools"])
			preloaded := splitList(f.data["skills"])
			canInvoke := len(tools) == 0 || contains(tools, "Skill")
			a := Agent{
				Name:            name,
				Scope:           src[0],
				Description:     str(f.data["description"]),
				Model:           str(f.data["model"]),
				Tools:           tools,
				PreloadedSkills: nonNil(preloaded),
				CanInvokeSkills: canInvoke,
				Prompt:          strings.TrimSpace(f.content),
				File:            f.file,
				Error:           f.err,
			}
			if a.Model == "" {
				a.Model = "inherit"
			}
			if len(a.Tools) == 0 {
				a.Tools = []string{"(all tools)"}
			}
			a.MissingSkills = []string{}
			for _, p := range preloaded {
				if !have[p] {
					a.MissingSkills = append(a.MissingSkills, p)
				}
			}
			a.DiscoverableSkills = []string{}
			if canInvoke {
				for _, n := range skillNames {
					if !contains(preloaded, n) {
						a.DiscoverableSkills = append(a.DiscoverableSkills, n)
					}
				}
			}
			byName[name] = a
		}
	}
	agents := make([]Agent, 0, len(byName)+1)
	for _, a := range byName {
		agents = append(agents, a)
	}
	sort.Slice(agents, func(i, j int) bool { return agents[i].Name < agents[j].Name })
	main := Agent{
		Name:               mainAgent,
		Scope:              "session",
		Description:        "Default Claude Code session in this scope. Can run /shipit and dispatch every agent.",
		Model:              "default",
		Tools:              []string{"(all tools)"},
		PreloadedSkills:    []string{},
		MissingSkills:      []string{},
		CanInvokeSkills:    true,
		DiscoverableSkills: nonNil(skillNames),
	}
	return append([]Agent{main}, agents...)
}

func listRuns(projectPath string) []ShipRun {
	dir := filepath.Join(projectPath, ".agent", "runs")
	entries, err := os.ReadDir(dir)
	if err != nil {
		return []ShipRun{}
	}
	runs := []ShipRun{}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		runDir := filepath.Join(dir, e.Name())
		r := ShipRun{ID: e.Name(), Files: []string{}}
		if b, err := os.ReadFile(filepath.Join(runDir, "state.json")); err == nil {
			_ = json.Unmarshal(b, &r.State)
		}
		if files, err := os.ReadDir(runDir); err == nil {
			for _, f := range files {
				if !strings.HasPrefix(f.Name(), ".") {
					r.Files = append(r.Files, f.Name())
				}
			}
		}
		if u := str(r.State["updated_at"]); u != "" {
			r.Updated = u
		} else if st, err := os.Stat(runDir); err == nil {
			r.Updated = st.ModTime().UTC().Format("2006-01-02T15:04:05.000Z")
		}
		runs = append(runs, r)
	}
	sort.Slice(runs, func(i, j int) bool { return runs[i].Updated > runs[j].Updated })
	return runs
}

func contains(xs []string, s string) bool {
	for _, x := range xs {
		if x == s {
			return true
		}
	}
	return false
}

func nonNil(xs []string) []string {
	if xs == nil {
		return []string{}
	}
	return xs
}

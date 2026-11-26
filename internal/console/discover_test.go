package console

import (
	"os"
	"path/filepath"
	"testing"
)

func write(t *testing.T, p, s string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(s), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestParseFrontmatter(t *testing.T) {
	data, body, err := parseFrontmatter([]byte("---\r\nname: x\r\ntools: Read, Skill\r\n---\r\nHello\r\n"))
	if err != nil || data["name"] != "x" || data["tools"] != "Read, Skill" || body != "Hello\n" {
		t.Fatalf("got %v %q %v", data, body, err)
	}
	data, body, _ = parseFrontmatter([]byte("no front matter"))
	if len(data) != 0 || body != "no front matter" {
		t.Fatalf("got %v %q", data, body)
	}
	if _, _, err := parseFrontmatter([]byte("---\nname: [oops\n---\n")); err == nil {
		t.Fatal("expected YAML error")
	}
}

func TestDiscoverAgentsAndSkills(t *testing.T) {
	home, proj := t.TempDir(), t.TempDir()
	write(t, filepath.Join(home, "skills", "shipit", "SKILL.md"), "---\nname: shipit\ndescription: user shipit\n---\n")
	write(t, filepath.Join(proj, ".claude", "skills", "shipit", "SKILL.md"), "---\nname: shipit\ndescription: project shipit\n---\n")
	write(t, filepath.Join(proj, ".claude", "skills", "lint", "SKILL.md"), "---\ndescription: lint it\n---\n")
	write(t, filepath.Join(proj, ".claude", "agents", "reviewer.md"), "---\nname: reviewer\nmodel: opus\ntools: Read, Grep\nskills:\n  - lint\n  - gone\n---\nReview code.\n")
	write(t, filepath.Join(home, "agents", "helper.md"), "---\ndescription: helps\n---\n")

	skills := discoverSkills(home, proj)
	if len(skills) != 2 || skills[0].Name != "lint" || skills[1].Description != "project shipit" || skills[1].Scope != "project" {
		t.Fatalf("skills = %+v", skills)
	}
	agents := discoverAgents(home, proj, skills)
	if len(agents) != 3 || agents[0].Name != mainAgent || agents[1].Name != "helper" || agents[2].Name != "reviewer" {
		t.Fatalf("agents = %+v", agents)
	}
	h, r := agents[1], agents[2]
	if h.Model != "inherit" || !h.CanInvokeSkills || len(h.DiscoverableSkills) != 2 || h.Tools[0] != "(all tools)" {
		t.Errorf("helper = %+v", h)
	}
	if r.Model != "opus" || r.CanInvokeSkills || len(r.DiscoverableSkills) != 0 || r.Prompt != "Review code." ||
		len(r.MissingSkills) != 1 || r.MissingSkills[0] != "gone" {
		t.Errorf("reviewer = %+v", r)
	}
}

func TestListRuns(t *testing.T) {
	proj := t.TempDir()
	write(t, filepath.Join(proj, ".agent", "runs", "a", "state.json"), `{"status":"running","updated_at":"2026-01-02"}`)
	write(t, filepath.Join(proj, ".agent", "runs", "a", "plan.md"), "plan")
	write(t, filepath.Join(proj, ".agent", "runs", "b", "state.json"), `{"status":"done","updated_at":"2026-01-01"}`)
	runs := listRuns(proj)
	if len(runs) != 2 || runs[0].ID != "a" || runs[0].State["status"] != "running" || len(runs[0].Files) != 2 {
		t.Fatalf("runs = %+v", runs)
	}
	if got := listRuns(t.TempDir()); got == nil || len(got) != 0 {
		t.Fatalf("no runs dir: %+v", got)
	}
}

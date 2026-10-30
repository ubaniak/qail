package git

import (
	"os"
	"path/filepath"
	"strings"
)

// CurrentBranch returns the branch checked out in the repo at dir, or
// the short commit hash when HEAD is detached. It reads .git/HEAD
// directly instead of shelling out, so listing every repo of every
// workspace stays cheap. Returns "" when dir is not a readable git repo.
func CurrentBranch(dir string) string {
	gitDir := filepath.Join(dir, ".git")
	// Worktrees and submodules have a .git file pointing elsewhere.
	if b, err := os.ReadFile(gitDir); err == nil {
		target, ok := strings.CutPrefix(strings.TrimSpace(string(b)), "gitdir:")
		if !ok {
			return ""
		}
		target = strings.TrimSpace(target)
		if !filepath.IsAbs(target) {
			target = filepath.Join(dir, target)
		}
		gitDir = target
	}
	b, err := os.ReadFile(filepath.Join(gitDir, "HEAD"))
	if err != nil {
		return ""
	}
	head := strings.TrimSpace(string(b))
	if ref, ok := strings.CutPrefix(head, "ref:"); ok {
		return strings.TrimPrefix(strings.TrimSpace(ref), "refs/heads/")
	}
	if len(head) > 7 {
		return head[:7]
	}
	return head
}

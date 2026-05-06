package git

import (
	"context"

	"github.com/ubaniak/qail/internal/forms"
	"github.com/ubaniak/qail/internal/runner"
)

// Runner is the narrow subprocess interface the git module needs.
// runner.OS satisfies it in production; runner.Recorder in tests.
type Runner interface {
	Run(ctx context.Context, cmd runner.Command) (runner.Result, error)
}

// Git wraps the git CLI behind a Runner seam.
type Git struct {
	r Runner
}

// New returns a Git wired to r.
func New(r Runner) *Git { return &Git{r: r} }

// Default returns a Git wired to the OS Runner. Use for non-test callers.
func Default() *Git { return New(runner.NewOS()) }

// Clone runs `git clone <repo> <path>` and returns combined stdout.
func (g *Git) Clone(repo, path string) (string, error) {
	res, err := g.r.Run(context.Background(), runner.Command{
		Name: "git",
		Args: []string{"clone", repo, path},
	})
	return string(res.Stdout), err
}

// CloneWithProgress wraps Clone in the huh spinner UI. Errors are swallowed
// to preserve the previous package-level behaviour; the spinner has no
// channel for them.
func (g *Git) CloneWithProgress(repo, path, message string) {
	clone := func() {
		g.Clone(repo, path)
	}
	forms.Spinner(clone, message)
}

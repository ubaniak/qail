package git

import (
	"context"

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

// Clone runs `git clone <repo> <path>` under ctx and returns combined
// stdout. Pure domain operation — UI concerns (progress spinners, log
// streaming) belong at the orchestration layer, not here.
func (g *Git) Clone(ctx context.Context, repo, path string) (string, error) {
	res, err := g.r.Run(ctx, runner.Command{
		Name: "git",
		Args: []string{"clone", repo, path},
	})
	return string(res.Stdout), err
}

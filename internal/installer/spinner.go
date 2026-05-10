package installer

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/ubaniak/qail/internal/forms"
	"github.com/ubaniak/qail/internal/git"
)

// spinnerGit wraps *git.Git in the huh spinner UI used during workspace
// creation from a TTY. Lives at the orchestration layer so the git package
// stays free of TUI dependencies; the writer arg is ignored — the spinner
// owns the output.
type spinnerGit struct{ inner *git.Git }

// Clone runs the underlying git clone inside a spinner. The spinner has no
// error channel, so the closure captures the error for return.
func (s spinnerGit) Clone(ctx context.Context, repo, path, message string, _ io.Writer) error {
	var err error
	forms.Spinner(func() { _, err = s.inner.Clone(ctx, repo, path) }, message)
	return err
}

// streamGit wraps *git.Git for non-TTY callers. It writes a one-line
// "Cloning..." message before the call and "done" / error after, so HTTP
// SSE streams stay informative without depending on huh.
type streamGit struct{ inner *git.Git }

func (s streamGit) Clone(ctx context.Context, repo, path, message string, w io.Writer) error {
	if w == nil {
		w = io.Discard
	}
	start := time.Now()
	fmt.Fprintf(w, "%s ...\n", message)
	out, err := s.inner.Clone(ctx, repo, path)
	if err != nil {
		fmt.Fprintf(w, "clone failed: %v\n", err)
		return err
	}
	if len(out) > 0 {
		fmt.Fprintln(w, out)
	}
	fmt.Fprintf(w, "done in %s\n", time.Since(start).Round(time.Millisecond))
	return nil
}

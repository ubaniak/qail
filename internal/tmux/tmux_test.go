package tmux

import (
	"context"
	"errors"
	"testing"

	"github.com/ubaniak/qail/internal/runner"
)

func TestSessionExistsParsesExitCode(t *testing.T) {
	rec := runner.NewRecorder().Respond(runner.Result{ExitCode: 0}, nil)
	tx := New(rec)

	exists, err := tx.SessionExists(context.Background(), "work")
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if !exists {
		t.Fatalf("expected SessionExists=true on exit 0")
	}
	if len(rec.Calls) != 1 {
		t.Fatalf("calls = %d, want 1", len(rec.Calls))
	}
	got := rec.LastCall()
	if got.Name != "tmux" {
		t.Fatalf("name = %q, want tmux", got.Name)
	}
	want := []string{"has-session", "-t", "work"}
	for i, a := range want {
		if got.Args[i] != a {
			t.Fatalf("args[%d] = %q, want %q", i, got.Args[i], a)
		}
	}
}

func TestListSessionsParsesNewlineFormat(t *testing.T) {
	rec := runner.NewRecorder().RespondOK([]byte("alpha\nbeta\ngamma\n"))
	tx := New(rec)

	got, err := tx.ListSessions(context.Background())
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	want := []string{"alpha", "beta", "gamma"}
	if len(got) != len(want) {
		t.Fatalf("sessions = %v, want %v", got, want)
	}
	for i, s := range want {
		if got[i] != s {
			t.Fatalf("sessions[%d] = %q, want %q", i, got[i], s)
		}
	}
}

func TestListSessionsEmptyStdoutReturnsEmptySlice(t *testing.T) {
	// tmux server up with zero sessions — list-sessions exits 0 with no
	// stdout. Must not become [""] (which the UI renders as a phantom row).
	rec := runner.NewRecorder().RespondOK([]byte(""))
	tx := New(rec)

	got, err := tx.ListSessions(context.Background())
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("sessions = %#v, want empty", got)
	}
}

func TestListSessionsNoServerRunningReturnsEmptySlice(t *testing.T) {
	// tmux exits 1 with "no server running on ..." on stderr when the
	// daemon isn't up. Treat as zero sessions, not an error.
	rec := runner.NewRecorder().Respond(
		runner.Result{Stderr: []byte("no server running on /tmp/tmux-501/default"), ExitCode: 1},
		errors.New("exit status 1"),
	)
	tx := New(rec)

	got, err := tx.ListSessions(context.Background())
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("sessions = %#v, want empty", got)
	}
}

func TestRemoveSessionInvokesKill(t *testing.T) {
	rec := runner.NewRecorder().Respond(runner.Result{ExitCode: 0}, nil)
	tx := New(rec)

	if err := tx.RemoveSession(context.Background(), "dead"); err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	got := rec.LastCall()
	want := []string{"kill-session", "-t", "dead"}
	for i, a := range want {
		if got.Args[i] != a {
			t.Fatalf("args[%d] = %q, want %q", i, got.Args[i], a)
		}
	}
}

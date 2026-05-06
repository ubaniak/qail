package runner

import (
	"context"
	"strings"
	"testing"
)

func TestOSRunCapturesStdout(t *testing.T) {
	r := NewOS()
	res, err := r.Run(context.Background(), Command{
		Name: "echo",
		Args: []string{"hello"},
	})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if res.ExitCode != 0 {
		t.Fatalf("exit code = %d, want 0", res.ExitCode)
	}
	if !strings.Contains(string(res.Stdout), "hello") {
		t.Fatalf("stdout = %q, want contains 'hello'", res.Stdout)
	}
}

func TestOSRunNonZeroExit(t *testing.T) {
	r := NewOS()
	res, err := r.Run(context.Background(), Command{
		Name: "sh",
		Args: []string{"-c", "exit 2"},
	})
	if err == nil {
		t.Fatalf("expected error for non-zero exit")
	}
	if res.ExitCode != 2 {
		t.Fatalf("exit code = %d, want 2", res.ExitCode)
	}
}

func TestRecorderRecordsCalls(t *testing.T) {
	rec := NewRecorder().RespondOK([]byte("line1\nline2\n"))
	res, err := rec.Run(context.Background(), Command{
		Name: "tmux",
		Args: []string{"list-sessions"},
	})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if string(res.Stdout) != "line1\nline2\n" {
		t.Fatalf("stdout = %q", res.Stdout)
	}
	if len(rec.Calls) != 1 {
		t.Fatalf("calls = %d, want 1", len(rec.Calls))
	}
	if rec.LastCall().Name != "tmux" {
		t.Fatalf("name = %q", rec.LastCall().Name)
	}
}

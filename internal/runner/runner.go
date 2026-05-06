// Package runner provides a Runner seam over external command execution.
//
// Every consumer (git, scripts, tmux, workspace) defines its own narrow Runner
// interface satisfied by the OS adapter in this package. Tests substitute the
// Recorder adapter to assert exact invocations without spawning subprocesses.
package runner

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os/exec"
)

// Command describes a subprocess invocation. Stdin/Stdout/Stderr are optional;
// when nil the OS adapter captures into Result. When set they stream directly,
// allowing interactive use cases (editors, bash output) to inherit os.Stdout.
type Command struct {
	Name   string
	Args   []string
	Dir    string
	Env    []string
	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer
}

// Result holds captured output from a Command. Stdout/Stderr are nil when the
// caller supplied their own writers (no capture happened).
type Result struct {
	Stdout   []byte
	Stderr   []byte
	ExitCode int
}

// OS is the production Runner adapter backed by os/exec.
type OS struct{}

// NewOS returns the production adapter.
func NewOS() *OS { return &OS{} }

// Run executes cmd via os/exec. Captures stdout/stderr only when the caller
// did not provide writers. ExitCode is populated from *exec.ExitError; other
// errors (e.g. binary missing) return ExitCode = -1.
func (OS) Run(ctx context.Context, cmd Command) (Result, error) {
	c := exec.CommandContext(ctx, cmd.Name, cmd.Args...)
	c.Dir = cmd.Dir
	c.Env = cmd.Env
	c.Stdin = cmd.Stdin

	var stdoutBuf, stderrBuf bytes.Buffer
	if cmd.Stdout != nil {
		c.Stdout = cmd.Stdout
	} else {
		c.Stdout = &stdoutBuf
	}
	if cmd.Stderr != nil {
		c.Stderr = cmd.Stderr
	} else {
		c.Stderr = &stderrBuf
	}

	err := c.Run()

	res := Result{ExitCode: -1}
	if cmd.Stdout == nil {
		res.Stdout = stdoutBuf.Bytes()
	}
	if cmd.Stderr == nil {
		res.Stderr = stderrBuf.Bytes()
	}

	if err == nil {
		res.ExitCode = 0
		return res, nil
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		res.ExitCode = exitErr.ExitCode()
	}
	return res, err
}

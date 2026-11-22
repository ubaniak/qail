package console

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

// This file drives the installed Claude Code CLI directly (ADR-0020). It
// speaks the same stdio protocol the Claude Agent SDK uses under the hood:
//
//	claude --input-format stream-json --output-format stream-json --verbose
//	       --permission-prompt-tool stdio ...
//
// We write one `user` message to stdin, read newline-delimited JSON
// messages from stdout, and answer the CLI's `control_request` messages
// (subtype `can_use_tool`) with `control_response` messages on stdin. That
// is what makes the console's permission cards work without the SDK.
// Once the `result` message arrives we close stdin and the CLI exits.

// ErrNoClaude means no Claude Code CLI could be found.
var ErrNoClaude = errors.New("claude CLI not found (install Claude Code, or set QAIL_CLAUDE to its path)")

// Decision answers one permission prompt.
type Decision struct {
	Allow bool
	// Message is shown to the model when the tool is denied.
	Message string
	// Interrupt stops the whole turn instead of letting the model carry on.
	Interrupt bool
}

// Message is one stream-json message from the CLI, kept raw so callers can
// pick out only the fields they need.
type Message struct {
	Type    string          `json:"type"`
	Subtype string          `json:"subtype,omitempty"`
	Raw     json.RawMessage `json:"-"`
}

// RunOptions configures one turn of a Claude Code session.
type RunOptions struct {
	// Claude is the CLI binary. Required.
	Claude string
	// Cwd is the directory the agent works in.
	Cwd string
	// Prompt is the user's message for this turn.
	Prompt string
	// Agent is a named subagent from .claude/agents to run as the main
	// thread; "" runs the default session.
	Agent string
	// Resume continues an earlier session by id.
	Resume string
	// PermissionMode is default, acceptEdits or plan.
	PermissionMode string
	// AppendSystemPrompt is added after Claude Code's own system prompt.
	AppendSystemPrompt string
	// Env is the full environment for the CLI. nil = os.Environ().
	Env []string
	// CanUseTool answers a permission prompt. It may block (the user is
	// deciding) and must return when ctx is done. nil denies everything.
	CanUseTool func(ctx context.Context, tool string, input json.RawMessage) Decision
	// OnMessage receives every message except control traffic, in order.
	OnMessage func(Message)
}

// Args returns the CLI arguments for o (exported for tests).
func (o RunOptions) Args() []string {
	args := []string{
		"--input-format", "stream-json",
		"--output-format", "stream-json",
		"--verbose",
		"--include-partial-messages",
		"--permission-prompt-tool", "stdio",
		// Same sources as Claude Code in a terminal: ~/.claude, the
		// project's .claude/ and its settings.local.json.
		"--setting-sources", "user,project,local",
		// Skills are always invocable, like the SDK's `skills: "all"`.
		"--allowedTools", "Skill",
	}
	if o.PermissionMode != "" {
		args = append(args, "--permission-mode", o.PermissionMode)
	}
	if o.Agent != "" {
		args = append(args, "--agent", o.Agent)
	}
	if o.Resume != "" {
		args = append(args, "--resume", o.Resume)
	}
	if o.AppendSystemPrompt != "" {
		args = append(args, "--append-system-prompt", o.AppendSystemPrompt)
	}
	return args
}

// Run runs one turn and returns when the CLI exits. Cancelling ctx stops
// the CLI (the session stays resumable). A nil error means the CLI sent
// its `result` message.
func Run(ctx context.Context, o RunOptions) error {
	if o.Claude == "" {
		return ErrNoClaude
	}
	cmd := exec.CommandContext(ctx, o.Claude, o.Args()...)
	cmd.Dir = o.Cwd
	cmd.Env = o.Env
	if cmd.Env == nil {
		cmd.Env = os.Environ()
	}
	cmd.Cancel = func() error {
		if runtime.GOOS == "windows" {
			return cmd.Process.Kill()
		}
		return cmd.Process.Signal(os.Interrupt)
	}
	cmd.WaitDelay = 3 * time.Second
	var stderr tailBuffer
	cmd.Stderr = &stderr

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start claude: %w", err)
	}

	w := &lineWriter{w: stdin}
	prompt, _ := json.Marshal(map[string]any{
		"type":               "user",
		"session_id":         "",
		"parent_tool_use_id": nil,
		"message": map[string]any{
			"role":    "user",
			"content": []map[string]string{{"type": "text", "text": o.Prompt}},
		},
	})
	_ = w.write(prompt)

	var (
		pending   sync.WaitGroup
		cancelsMu sync.Mutex
		cancels   = map[string]context.CancelFunc{}
		gotResult bool
	)
	r := bufio.NewReaderSize(stdout, 1<<20)
	for {
		line, err := r.ReadBytes('\n')
		if len(bytes.TrimSpace(line)) > 0 {
			var head struct {
				Type      string          `json:"type"`
				Subtype   string          `json:"subtype"`
				RequestID string          `json:"request_id"`
				Request   json.RawMessage `json:"request"`
			}
			if json.Unmarshal(line, &head) != nil {
				// Not JSON (a stray log line); ignore it.
			} else {
				switch head.Type {
				case "control_request":
					reqCtx, cancel := context.WithCancel(ctx)
					cancelsMu.Lock()
					cancels[head.RequestID] = cancel
					cancelsMu.Unlock()
					pending.Add(1)
					go func(id string, req json.RawMessage) {
						defer pending.Done()
						resp := answerControl(reqCtx, o, req)
						cancelsMu.Lock()
						delete(cancels, id)
						cancelsMu.Unlock()
						cancel()
						b, _ := json.Marshal(map[string]any{
							"type":     "control_response",
							"response": controlResponse(id, resp),
						})
						_ = w.write(b)
					}(head.RequestID, head.Request)
				case "control_cancel_request":
					cancelsMu.Lock()
					if c := cancels[head.RequestID]; c != nil {
						c()
					}
					cancelsMu.Unlock()
				case "control_response", "keep_alive":
					// Replies to requests we never send; nothing to do.
				default:
					if o.OnMessage != nil {
						o.OnMessage(Message{Type: head.Type, Subtype: head.Subtype, Raw: append(json.RawMessage(nil), bytes.TrimSpace(line)...)})
					}
					if head.Type == "result" {
						gotResult = true
						// One prompt per process: EOF on stdin lets it exit.
						_ = w.close()
					}
				}
			}
		}
		if err != nil {
			break
		}
	}
	cancelsMu.Lock()
	for _, c := range cancels {
		c()
	}
	cancelsMu.Unlock()
	pending.Wait()
	_ = w.close()
	waitErr := cmd.Wait()

	switch {
	case ctx.Err() != nil:
		return ctx.Err()
	case gotResult:
		return nil
	case waitErr != nil:
		if msg := strings.TrimSpace(stderr.String()); msg != "" {
			return fmt.Errorf("claude: %v: %s", waitErr, msg)
		}
		return fmt.Errorf("claude: %w", waitErr)
	default:
		if msg := strings.TrimSpace(stderr.String()); msg != "" {
			return fmt.Errorf("claude exited without a result: %s", msg)
		}
		return errors.New("claude exited without a result")
	}
}

// answerControl handles one control_request. Only can_use_tool is
// expected with our flags; anything else gets an error reply so the CLI
// doesn't wait on it.
func answerControl(ctx context.Context, o RunOptions, raw json.RawMessage) any {
	var req struct {
		Subtype  string          `json:"subtype"`
		ToolName string          `json:"tool_name"`
		Input    json.RawMessage `json:"input"`
	}
	if err := json.Unmarshal(raw, &req); err != nil {
		return controlError(err.Error())
	}
	if req.Subtype != "can_use_tool" {
		return controlError("unsupported control request: " + req.Subtype)
	}
	input := req.Input
	if len(input) == 0 {
		input = json.RawMessage("{}")
	}
	d := Decision{Message: "No permission handler"}
	if o.CanUseTool != nil {
		d = o.CanUseTool(ctx, req.ToolName, input)
	}
	if d.Allow {
		return map[string]any{"behavior": "allow", "updatedInput": input}
	}
	msg := d.Message
	if msg == "" {
		msg = "Denied"
	}
	out := map[string]any{"behavior": "deny", "message": msg}
	if d.Interrupt {
		out["interrupt"] = true
	}
	return out
}

type controlErr struct{ msg string }

func controlError(msg string) any { return controlErr{msg} }

func controlResponse(id string, resp any) map[string]any {
	if e, ok := resp.(controlErr); ok {
		return map[string]any{"subtype": "error", "request_id": id, "error": e.msg}
	}
	return map[string]any{"subtype": "success", "request_id": id, "response": resp}
}

// lineWriter serialises newline-terminated writes to the CLI's stdin.
type lineWriter struct {
	mu     sync.Mutex
	w      io.WriteCloser
	closed bool
}

func (l *lineWriter) write(b []byte) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.closed {
		return io.ErrClosedPipe
	}
	_, err := l.w.Write(append(b, '\n'))
	return err
}

func (l *lineWriter) close() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.closed {
		return nil
	}
	l.closed = true
	return l.w.Close()
}

// tailBuffer keeps the last 4 KiB written to it (the CLI's stderr).
type tailBuffer struct {
	mu  sync.Mutex
	buf []byte
}

func (t *tailBuffer) Write(p []byte) (int, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.buf = append(t.buf, p...)
	if len(t.buf) > 4096 {
		t.buf = t.buf[len(t.buf)-4096:]
	}
	return len(p), nil
}

func (t *tailBuffer) String() string {
	t.mu.Lock()
	defer t.mu.Unlock()
	return string(t.buf)
}

// FindClaude resolves the Claude Code CLI: explicit, $QAIL_CLAUDE, PATH,
// then the usual install locations, since a menubar app does not inherit
// the shell's PATH.
func FindClaude(explicit string) (string, error) {
	for _, c := range []string{explicit, os.Getenv("QAIL_CLAUDE")} {
		if c != "" && fileExists(c) {
			return c, nil
		}
	}
	if p, err := exec.LookPath("claude"); err == nil {
		return p, nil
	}
	for _, c := range claudeCandidates() {
		if fileExists(c) {
			return c, nil
		}
	}
	return "", ErrNoClaude
}

func claudeCandidates() []string {
	home, _ := os.UserHomeDir()
	if runtime.GOOS == "windows" {
		appData := os.Getenv("APPDATA")
		return []string{
			filepath.Join(home, ".local", "bin", "claude.exe"),
			filepath.Join(home, ".claude", "local", "claude.exe"),
			filepath.Join(appData, "npm", "claude.cmd"),
		}
	}
	out := []string{
		filepath.Join(home, ".local", "bin", "claude"),
		filepath.Join(home, ".claude", "local", "claude"),
		"/opt/homebrew/bin/claude",
		"/usr/local/bin/claude",
		filepath.Join(home, ".npm-global", "bin", "claude"),
		filepath.Join(home, ".volta", "bin", "claude"),
		filepath.Join(home, ".bun", "bin", "claude"),
	}
	for _, n := range nvmBins(filepath.Join(home, ".nvm", "versions", "node")) {
		out = append(out, filepath.Join(n, "claude"))
	}
	return out
}

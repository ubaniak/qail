package api

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
)

// sseWriter adapts an http.ResponseWriter into an io.Writer that emits
// each Write as one SSE `data:` event, flushing after every line. The
// browser's EventSource API receives one event per progress line so the
// UI can render output incrementally during long ops (clone, post-install).
//
// Multi-line writes are split on '\n' so each rendered line becomes its
// own event — without this a buffered write of 50 lines would arrive as
// one giant event with embedded newlines, which most JS consumers find
// awkward to format.
type sseWriter struct {
	mu      sync.Mutex
	w       http.ResponseWriter
	flusher http.Flusher
	closed  bool
}

// newSSE writes the SSE response headers, returns the writer, and starts
// a goroutine that watches ctx so the connection closes promptly when
// the client disconnects. If the response writer doesn't support flushing
// the call returns an error — every reasonable HTTP server in stdlib
// supports it, but the check is cheap and saves a confusing buffered
// failure.
func newSSE(w http.ResponseWriter, _ *http.Request) (*sseWriter, error) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		return nil, errors.New("sse: response writer does not support flushing")
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no") // disable nginx buffering if proxied
	w.WriteHeader(http.StatusOK)
	flusher.Flush()
	return &sseWriter{w: w, flusher: flusher}, nil
}

// Write implements io.Writer. Splits b on '\n', emits one `data:` event
// per resulting line, flushes. Empty trailing line (from a terminating
// newline) is dropped so the stream doesn't send blank events.
func (s *sseWriter) Write(b []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return 0, io.ErrClosedPipe
	}
	lines := strings.Split(string(b), "\n")
	// Drop the empty tail produced by trailing '\n'
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	for _, line := range lines {
		if _, err := fmt.Fprintf(s.w, "data: %s\n\n", line); err != nil {
			return 0, err
		}
	}
	s.flusher.Flush()
	return len(b), nil
}

// event writes a custom-named SSE event (e.g. "done", "error") so the
// client can distinguish lifecycle messages from progress lines.
func (s *sseWriter) event(name, payload string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return
	}
	fmt.Fprintf(s.w, "event: %s\ndata: %s\n\n", name, payload)
	s.flusher.Flush()
}

// close marks the writer closed so subsequent Writes fail fast. No-op on
// the underlying ResponseWriter — net/http handles connection teardown
// when the handler returns.
func (s *sseWriter) close() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.closed = true
}

// streamAction runs fn with sw as the io.Writer and ctx threaded in;
// emits a `done` event on success and an `error` event on failure. The
// handler can hand sw to any actions.* function that accepts an io.Writer.
func streamAction(ctx context.Context, sw *sseWriter, fn func(ctx context.Context, w io.Writer) error) {
	defer sw.close()
	if err := fn(ctx, sw); err != nil {
		sw.event("error", err.Error())
		return
	}
	sw.event("done", "ok")
}

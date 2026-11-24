package app

// HTTP transport for Bindings, used by `qail tray` (ADR-0021). The same
// React UI that Electron loads talks to it from a browser tab:
//
//	POST /api/call            {"method":"AddRepo","args":[...]}
//	                          → {"result":...} or {"error":"..."}
//	GET  /api/events          server-sent events, one JSON {event,data} per message
//	POST /api/pick-directory  {"title":"..."} → {"path":"..."} ("" when cancelled)
//	GET  /api/health          {"status":"ok","app":"qail-tray"}
//	GET  /                    the embedded UI
//
// Like the stdio bridge, errors from a Bindings method come back as data
// with status 200, so the frontend handles both transports the same way.
//
// Security: the server is meant to listen on 127.0.0.1 only. There is no
// token; instead guardLocal refuses any Host other than loopback (DNS
// rebinding), any cross-origin API call, and API POSTs that aren't JSON
// (which a cross-site form or no-cors fetch could otherwise send without a
// CORS preflight). Other programs running as the user can still call the
// API, as they could run qail itself.

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"mime"
	"net"
	"net/http"
	"regexp"
	"strings"
	"sync"

	"github.com/ubaniak/qail/internal/config"
)

// HTTPAppName identifies a qail tray server in /api/health, so a second
// `qail tray` can tell its own kind from something else on the port.
const HTTPAppName = "qail-tray"

// maxCallBody bounds a /api/call body. Script contents travel through
// WriteScript, matching the stdio bridge's line limit.
const maxCallBody = 16 << 20

// DirectoryPicker shows a native "choose folder" dialog. It returns ""
// when the user cancels.
type DirectoryPicker func(ctx context.Context, title string) (string, error)

// HTTPServer serves Bindings and the UI over HTTP.
type HTTPServer struct {
	b      *Bindings
	ui     fs.FS
	pick   DirectoryPicker
	events *eventHub
}

// NewHTTPServer wires a Bindings to an HTTP handler. ui holds the built
// frontend (index.html at its root); pick may be nil, in which case the
// UI's Browse button reports that it is unavailable.
func NewHTTPServer(ctx context.Context, s config.Store, ui fs.FS, pick DirectoryPicker, opts ...ServerOption) *HTTPServer {
	h := &HTTPServer{ui: ui, pick: pick, events: newEventHub()}
	h.b = New(ctx, s, h.events.emit)
	for _, o := range opts {
		o(h.b)
	}
	return h
}

// Bindings returns the method set behind the server, for in-process
// callers such as the tray menu.
func (h *HTTPServer) Bindings() *Bindings { return h.b }

// Handler returns the guarded HTTP handler.
func (h *HTTPServer) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/health", func(w http.ResponseWriter, _ *http.Request) {
		writeJSONResponse(w, http.StatusOK, map[string]string{"status": "ok", "app": HTTPAppName})
	})
	mux.HandleFunc("POST /api/call", h.handleCall)
	mux.HandleFunc("GET /api/events", h.handleEvents)
	mux.HandleFunc("POST /api/pick-directory", h.handlePickDirectory)
	ui := h.uiHandler()
	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/") {
			writeJSONResponse(w, http.StatusNotFound, map[string]string{"error": "not found"})
			return
		}
		ui.ServeHTTP(w, r)
	})
	return guardLocal(mux)
}

type callRequest struct {
	Method string            `json:"method"`
	Args   []json.RawMessage `json:"args"`
}

type callResponse struct {
	Result any    `json:"result"`
	Error  string `json:"error,omitempty"`
}

func (h *HTTPServer) handleCall(w http.ResponseWriter, r *http.Request) {
	var req callRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxCallBody)).Decode(&req); err != nil {
		writeJSONResponse(w, http.StatusBadRequest, callResponse{Error: fmt.Sprintf("invalid request: %v", err)})
		return
	}
	if req.Args == nil {
		req.Args = []json.RawMessage{}
	}
	result, err := dispatch(h.b, req.Method, req.Args)
	resp := callResponse{Result: result}
	if err != nil {
		resp.Error = err.Error()
	}
	writeJSONResponse(w, http.StatusOK, resp)
}

func (h *HTTPServer) handleEvents(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("X-Accel-Buffering", "no")
	ch, unsubscribe := h.events.subscribe()
	defer unsubscribe()

	w.WriteHeader(http.StatusOK)
	// A comment line so the browser's EventSource sees the stream open.
	fmt.Fprint(w, ": connected\n\n")
	flusher.Flush()
	for {
		select {
		case <-r.Context().Done():
			return
		case msg := <-ch:
			fmt.Fprintf(w, "data: %s\n\n", msg)
			flusher.Flush()
		}
	}
}

func (h *HTTPServer) handlePickDirectory(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Title string `json:"title"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 64<<10)).Decode(&req); err != nil {
		writeJSONResponse(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
		return
	}
	if h.pick == nil {
		writeJSONResponse(w, http.StatusOK, map[string]string{"error": "folder picker not available; type the path instead"})
		return
	}
	path, err := h.pick(r.Context(), req.Title)
	if err != nil {
		writeJSONResponse(w, http.StatusOK, map[string]string{"error": err.Error()})
		return
	}
	writeJSONResponse(w, http.StatusOK, map[string]string{"path": path})
}

// notBuiltPage is served when the binary was built without the frontend
// (plain `make build`), so the tray still explains itself.
const notBuiltPage = `<!doctype html><meta charset="utf-8"><title>qail</title>
<body style="font-family:system-ui;background:#09090b;color:#e4e4e7;padding:2rem">
<h1>qail UI not built</h1>
<p>This qail binary was built without the desktop UI. Run <code>make tray</code>
(or <code>make frontend build</code>) and start <code>qail tray</code> again.</p>`

func (h *HTTPServer) uiHandler() http.Handler {
	if h.ui != nil {
		if _, err := fs.Stat(h.ui, "index.html"); err == nil {
			files := http.FileServerFS(h.ui)
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// The UI is one page; always load fresh after an upgrade.
				if r.URL.Path == "/" || r.URL.Path == "/index.html" {
					w.Header().Set("Cache-Control", "no-cache")
				}
				files.ServeHTTP(w, r)
			})
		}
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, notBuiltPage)
	})
}

var loopbackOrigin = regexp.MustCompile(`^http://(127\.0\.0\.1|localhost|\[::1\]):\d+$`)

func isLoopbackHost(hostport string) bool {
	host, _, err := net.SplitHostPort(hostport)
	if err != nil {
		host = hostport
	}
	host = strings.Trim(host, "[]")
	return host == "127.0.0.1" || host == "localhost" || host == "::1"
}

// guardLocal keeps web pages in the user's browser away from the API. See
// the package comment at the top of this file.
func guardLocal(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !isLoopbackHost(r.Host) {
			writeJSONResponse(w, http.StatusForbidden, map[string]string{"error": "host not allowed"})
			return
		}
		if strings.HasPrefix(r.URL.Path, "/api/") {
			if o := r.Header.Get("Origin"); o != "" && !loopbackOrigin.MatchString(o) {
				writeJSONResponse(w, http.StatusForbidden, map[string]string{"error": "cross-origin blocked"})
				return
			}
			if r.Header.Get("Sec-Fetch-Site") == "cross-site" {
				writeJSONResponse(w, http.StatusForbidden, map[string]string{"error": "cross-origin blocked"})
				return
			}
			if r.Method == http.MethodPost {
				ct, _, _ := mime.ParseMediaType(r.Header.Get("Content-Type"))
				if ct != "application/json" {
					writeJSONResponse(w, http.StatusUnsupportedMediaType, map[string]string{"error": "expected application/json"})
					return
				}
			}
		}
		next.ServeHTTP(w, r)
	})
}

func writeJSONResponse(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// eventHub fans Bindings events out to every open /api/events stream.
// Events emitted while no browser tab is listening are dropped, as they
// are in Electron while the window is closed.
type eventHub struct {
	mu   sync.Mutex
	subs map[chan []byte]struct{}
}

func newEventHub() *eventHub { return &eventHub{subs: map[chan []byte]struct{}{}} }

func (e *eventHub) emit(name string, data any) {
	msg, err := json.Marshal(rpcEvent{Event: name, Data: data})
	if err != nil {
		return
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	for ch := range e.subs {
		select {
		case ch <- msg:
		default: // a stalled tab loses progress lines rather than blocking a clone
		}
	}
}

func (e *eventHub) subscribe() (<-chan []byte, func()) {
	ch := make(chan []byte, 256)
	e.mu.Lock()
	e.subs[ch] = struct{}{}
	e.mu.Unlock()
	return ch, func() {
		e.mu.Lock()
		delete(e.subs, ch)
		e.mu.Unlock()
	}
}

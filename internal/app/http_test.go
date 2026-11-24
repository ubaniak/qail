package app

import (
	"bufio"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"github.com/ubaniak/qail/internal/config"
)

func newTestHTTP(t *testing.T, pick DirectoryPicker) (*HTTPServer, *httptest.Server) {
	t.Helper()
	ui := fstest.MapFS{"index.html": {Data: []byte("<html>qail</html>")}}
	h := NewHTTPServer(context.Background(), config.NewMemoryStore(), ui, pick)
	ts := httptest.NewServer(h.Handler())
	t.Cleanup(ts.Close)
	return h, ts
}

func post(t *testing.T, url, body string, hdr map[string]string) (*http.Response, map[string]json.RawMessage) {
	t.Helper()
	req, _ := http.NewRequest(http.MethodPost, url, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	for k, v := range hdr {
		req.Header.Set(k, v)
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	var out map[string]json.RawMessage
	_ = json.NewDecoder(res.Body).Decode(&out)
	return res, out
}

func TestHTTPCallsBindings(t *testing.T) {
	_, ts := newTestHTTP(t, nil)

	res, out := post(t, ts.URL+"/api/call", `{"method":"AddRepo","args":["api","git@example.com:api.git"]}`, nil)
	if res.StatusCode != 200 || out["error"] != nil {
		t.Fatalf("AddRepo: %d %s", res.StatusCode, out["error"])
	}
	_, out = post(t, ts.URL+"/api/call", `{"method":"ListRepos","args":[]}`, nil)
	if !strings.Contains(string(out["result"]), "git@example.com:api.git") {
		t.Fatalf("ListRepos result = %s", out["result"])
	}

	// Method errors come back as data, like the stdio bridge.
	res, out = post(t, ts.URL+"/api/call", `{"method":"Nope","args":[]}`, nil)
	if res.StatusCode != 200 || !strings.Contains(string(out["error"]), "unknown method") {
		t.Fatalf("unknown method: %d %s", res.StatusCode, out["error"])
	}
}

func TestHTTPGuard(t *testing.T) {
	_, ts := newTestHTTP(t, nil)
	call := `{"method":"ListRepos","args":[]}`

	cases := []struct {
		name string
		hdr  map[string]string
		ct   string
		want int
	}{
		{"same origin", map[string]string{"Origin": "http://127.0.0.1:4318"}, "application/json", 200},
		{"localhost origin", map[string]string{"Origin": "http://localhost:4318"}, "application/json", 200},
		{"other site", map[string]string{"Origin": "https://evil.example"}, "application/json", 403},
		{"cross-site fetch metadata", map[string]string{"Sec-Fetch-Site": "cross-site"}, "application/json", 403},
		{"rebound host", map[string]string{"Host": "evil.example:4318"}, "application/json", 403},
		{"form post", nil, "text/plain", 415},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			req, _ := http.NewRequest(http.MethodPost, ts.URL+"/api/call", strings.NewReader(call))
			req.Header.Set("Content-Type", c.ct)
			for k, v := range c.hdr {
				if k == "Host" {
					req.Host = v
					continue
				}
				req.Header.Set(k, v)
			}
			res, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatal(err)
			}
			res.Body.Close()
			if res.StatusCode != c.want {
				t.Fatalf("status = %d, want %d", res.StatusCode, c.want)
			}
		})
	}
}

func TestHTTPServesUI(t *testing.T) {
	_, ts := newTestHTTP(t, nil)
	res, err := http.Get(ts.URL + "/")
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		t.Fatalf("status = %d", res.StatusCode)
	}

	// Without a built frontend the tray explains instead of 404ing.
	h := NewHTTPServer(context.Background(), config.NewMemoryStore(), fstest.MapFS{}, nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "http://127.0.0.1/", nil)
	h.Handler().ServeHTTP(rec, req)
	if !strings.Contains(rec.Body.String(), "UI not built") {
		t.Fatalf("placeholder page missing: %q", rec.Body.String())
	}
}

func TestHTTPEvents(t *testing.T) {
	h, ts := newTestHTTP(t, nil)
	res, err := http.Get(ts.URL + "/api/events")
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	r := bufio.NewReader(res.Body)
	if line, _ := r.ReadString('\n'); !strings.HasPrefix(line, ": connected") {
		t.Fatalf("first line = %q", line)
	}
	_, _ = r.ReadString('\n')

	// The subscription is registered just before ": connected" is sent.
	h.b.emit("workspace:progress", "cloning api")
	done := make(chan string, 1)
	go func() {
		line, _ := r.ReadString('\n')
		done <- line
	}()
	select {
	case line := <-done:
		if line != `data: {"event":"workspace:progress","data":"cloning api"}`+"\n" {
			t.Fatalf("event line = %q", line)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("no event")
	}
}

func TestHTTPPickDirectory(t *testing.T) {
	_, ts := newTestHTTP(t, func(_ context.Context, title string) (string, error) {
		return "/picked/" + title, nil
	})
	_, out := post(t, ts.URL+"/api/pick-directory", `{"title":"root"}`, nil)
	if string(out["path"]) != `"/picked/root"` {
		t.Fatalf("path = %s", out["path"])
	}

	_, ts = newTestHTTP(t, nil)
	_, out = post(t, ts.URL+"/api/pick-directory", `{"title":"root"}`, nil)
	if out["error"] == nil {
		t.Fatal("want an error without a picker")
	}
}

package console

import (
	"bufio"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/ubaniak/qail/internal/config"
)

// fakeClaude writes a shell script that plays the CLI's side of the
// stream-json protocol: it reads the prompt, asks permission for a Bash
// call, then reports whether it was allowed. Its argv and the control
// response it got are saved next to it.
func fakeClaude(t *testing.T) (bin, dir string) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("fake claude is a shell script")
	}
	dir = t.TempDir()
	bin = filepath.Join(dir, "claude")
	script := `#!/bin/sh
printf '%s\n' "$@" > "` + dir + `/args"
read -r prompt
printf '%s\n' "$prompt" > "` + dir + `/prompt"
echo '{"type":"system","subtype":"init","session_id":"sess-1","model":"claude-test","skills":["ship"],"tools":["Bash"],"permissionMode":"default"}'
echo 'not json, a stray log line'
echo '{"type":"stream_event","parent_tool_use_id":null,"event":{"type":"content_block_delta","delta":{"type":"text_delta","text":"Working"}}}'
echo '{"type":"assistant","parent_tool_use_id":null,"message":{"content":[{"type":"tool_use","id":"tu1","name":"Bash","input":{"command":"touch x"}}]}}'
echo '{"type":"control_request","request_id":"req-1","request":{"subtype":"can_use_tool","tool_name":"Bash","input":{"command":"touch x"}}}'
read -r answer
printf '%s\n' "$answer" > "` + dir + `/answer"
case "$answer" in
  *'"behavior":"allow"'*) echo '{"type":"user","message":{"content":[{"type":"tool_result","tool_use_id":"tu1","content":"ok","is_error":false}]}}'
     echo '{"type":"assistant","parent_tool_use_id":null,"message":{"content":[{"type":"text","text":"Done."}]}}' ;;
  *) echo '{"type":"assistant","parent_tool_use_id":null,"message":{"content":[{"type":"text","text":"Denied."}]}}' ;;
esac
echo '{"type":"result","subtype":"success","total_cost_usd":0.01,"num_turns":2,"duration_ms":5,"is_error":false}'
cat > /dev/null
`
	if err := os.WriteFile(bin, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	return bin, dir
}

func TestRunAnswersPermissionPrompts(t *testing.T) {
	bin, dir := fakeClaude(t)
	var types []string
	var asked string
	err := Run(context.Background(), RunOptions{
		Claude:         bin,
		Cwd:            dir,
		Prompt:         "make x",
		Agent:          "reviewer",
		Resume:         "sess-0",
		PermissionMode: "acceptEdits",
		CanUseTool: func(_ context.Context, tool string, input json.RawMessage) Decision {
			asked = tool + " " + string(input)
			return Decision{Allow: true}
		},
		OnMessage: func(m Message) { types = append(types, m.Type) },
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if asked != `Bash {"command":"touch x"}` {
		t.Errorf("CanUseTool got %q", asked)
	}
	if got := strings.Join(types, ","); got != "system,stream_event,assistant,user,assistant,result" {
		t.Errorf("messages = %s", got)
	}

	var answer struct {
		Type     string `json:"type"`
		Response struct {
			Subtype   string `json:"subtype"`
			RequestID string `json:"request_id"`
			Response  struct {
				Behavior     string          `json:"behavior"`
				UpdatedInput json.RawMessage `json:"updatedInput"`
			} `json:"response"`
		} `json:"response"`
	}
	b, _ := os.ReadFile(filepath.Join(dir, "answer"))
	if err := json.Unmarshal(b, &answer); err != nil {
		t.Fatalf("answer %q: %v", b, err)
	}
	if answer.Type != "control_response" || answer.Response.Subtype != "success" || answer.Response.RequestID != "req-1" ||
		answer.Response.Response.Behavior != "allow" || string(answer.Response.Response.UpdatedInput) != `{"command":"touch x"}` {
		t.Errorf("answer = %s", b)
	}

	var prompt struct {
		Type    string `json:"type"`
		Message struct {
			Role    string `json:"role"`
			Content []struct{ Text string } `json:"content"`
		} `json:"message"`
	}
	b, _ = os.ReadFile(filepath.Join(dir, "prompt"))
	if json.Unmarshal(b, &prompt) != nil || prompt.Type != "user" || prompt.Message.Content[0].Text != "make x" {
		t.Errorf("prompt = %s", b)
	}

	args, _ := os.ReadFile(filepath.Join(dir, "args"))
	for _, want := range []string{"--permission-prompt-tool\nstdio", "--input-format\nstream-json", "--agent\nreviewer", "--resume\nsess-0", "--permission-mode\nacceptEdits", "--setting-sources\nuser,project,local"} {
		if !strings.Contains(string(args), want) {
			t.Errorf("args missing %q:\n%s", want, args)
		}
	}
}

func TestRunDeny(t *testing.T) {
	bin, dir := fakeClaude(t)
	err := Run(context.Background(), RunOptions{
		Claude: bin, Cwd: dir, Prompt: "x",
		CanUseTool: func(context.Context, string, json.RawMessage) Decision {
			return Decision{Message: "nope", Interrupt: true}
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(filepath.Join(dir, "answer"))
	if !strings.Contains(string(b), `"behavior":"deny"`) || !strings.Contains(string(b), `"message":"nope"`) || !strings.Contains(string(b), `"interrupt":true`) {
		t.Errorf("answer = %s", b)
	}
}

func TestRunCancelStopsWaitingPrompt(t *testing.T) {
	bin, dir := fakeClaude(t)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		done <- Run(ctx, RunOptions{
			Claude: bin, Cwd: dir, Prompt: "x",
			CanUseTool: func(ctx context.Context, _ string, _ json.RawMessage) Decision {
				cancel() // the user hits Stop while the card is open
				<-ctx.Done()
				return Decision{Message: "Aborted", Interrupt: true}
			},
		})
	}()
	select {
	case err := <-done:
		if err == nil {
			t.Fatal("expected an error after cancel")
		}
	case <-time.After(10 * time.Second):
		t.Fatal("Run did not return after cancel")
	}
}

func TestRunReportsMissingResult(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell script")
	}
	bin := filepath.Join(t.TempDir(), "claude")
	os.WriteFile(bin, []byte("#!/bin/sh\necho 'Invalid API key' >&2\nexit 1\n"), 0o755)
	err := Run(context.Background(), RunOptions{Claude: bin, Prompt: "x"})
	if err == nil || !strings.Contains(err.Error(), "Invalid API key") {
		t.Fatalf("got %v", err)
	}
}

// TestChatEndToEnd drives the console's HTTP API the way the browser
// does: send a message, answer the permission card, read the SSE events,
// and check the thread was saved with the session id for --resume.
func TestChatEndToEnd(t *testing.T) {
	bin, _ := fakeClaude(t)
	root := t.TempDir()
	os.MkdirAll(filepath.Join(root, "pay", "api"), 0o755)
	store := config.NewMemoryStoreFrom(config.Config{
		Root:       root,
		Repos:      map[string]string{"api": "git@x:api.git", "web": "git@x:web.git"},
		Workspaces: config.Workspace{"pay": config.WorkspaceProfile{Repos: []string{"api", "web"}}},
	})
	data := t.TempDir()
	app, err := newServer(Options{Store: store, DataDir: data, Claude: bin, Log: os.Stderr})
	if err != nil {
		t.Fatal(err)
	}
	ts := httptest.NewServer(app.handler())
	defer ts.Close()

	res, err := http.Post(ts.URL+"/api/scope/pay/api/threads/main/chat", "application/json", strings.NewReader(`{"message":"make x","mode":"default"}`))
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	var events []string
	sc := bufio.NewScanner(res.Body)
	for sc.Scan() {
		line := sc.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		var ev map[string]any
		if err := json.Unmarshal([]byte(line[6:]), &ev); err != nil {
			t.Fatalf("bad event %q", line)
		}
		events = append(events, ev["type"].(string))
		if ev["type"] == "permission" {
			r, err := http.Post(ts.URL+"/api/permissions/"+ev["permId"].(string), "application/json", strings.NewReader(`{"allow":true}`))
			if err != nil || r.StatusCode != 200 {
				t.Fatalf("permission answer: %v %v", err, r.Status)
			}
			r.Body.Close()
		}
	}
	want := "init,delta,tool,permission,tool_result,text_done,result,done"
	if got := strings.Join(events, ","); got != want {
		t.Fatalf("events = %s, want %s", got, want)
	}

	th := app.threads.get(threadKey("pay/api", "main"))
	if th == nil || th.SessionID != "sess-1" || len(th.Messages) != 2 || th.Messages[1].Text != "Done." || th.Messages[1].Cost == nil {
		t.Fatalf("thread = %+v", th)
	}
	// Persisted in the Node console's format.
	b, _ := os.ReadFile(filepath.Join(data, "store.json"))
	if !strings.Contains(string(b), `"pay/api::main"`) || !strings.Contains(string(b), `"sessionId": "sess-1"`) {
		t.Fatalf("store.json = %s", b)
	}
	reopened, err := openThreadStore(data)
	if err != nil || reopened.get(threadKey("pay/api", "main")).SessionID != "sess-1" {
		t.Fatalf("reload: %v", err)
	}

	// A repo that isn't cloned can't chat.
	res2, _ := http.Post(ts.URL+"/api/scope/pay/web/threads/main/chat", "application/json", strings.NewReader(`{"message":"hi"}`))
	if res2.StatusCode != http.StatusConflict {
		t.Errorf("uncloned repo chat: %s", res2.Status)
	}
	res2.Body.Close()
}

func TestGuardBlocksOtherOriginsAndHosts(t *testing.T) {
	app, _ := newServer(Options{Store: config.NewMemoryStore()})
	h := app.handler()
	cases := []struct {
		host, origin string
		want         int
	}{
		{"127.0.0.1:4317", "", 200},
		{"localhost:4317", "http://localhost:4317", 200},
		{"127.0.0.1:4317", "https://evil.example", 403},
		{"evil.example:4317", "", 403}, // DNS rebinding
	}
	for _, c := range cases {
		req := httptest.NewRequest("GET", "/api/health", nil)
		req.Host = c.host
		if c.origin != "" {
			req.Header.Set("Origin", c.origin)
		}
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if rec.Code != c.want {
			t.Errorf("host %s origin %q: %d, want %d", c.host, c.origin, rec.Code, c.want)
		}
	}
}

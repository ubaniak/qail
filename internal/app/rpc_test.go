package app

import (
	"bytes"
	"context"
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/ubaniak/qail/internal/config"
)

// roundTrip feeds requests (one per line) through a fresh Server backed by
// s and returns the decoded response lines keyed by id.
func roundTrip(t *testing.T, s config.Store, requests ...string) map[uint64]map[string]json.RawMessage {
	t.Helper()
	var out bytes.Buffer
	srv := NewServer(context.Background(), s, &out)
	if err := srv.Serve(strings.NewReader(strings.Join(requests, "\n") + "\n")); err != nil {
		t.Fatalf("Serve: %v", err)
	}
	got := map[uint64]map[string]json.RawMessage{}
	for _, line := range strings.Split(strings.TrimSpace(out.String()), "\n") {
		var m map[string]json.RawMessage
		if err := json.Unmarshal([]byte(line), &m); err != nil {
			t.Fatalf("bad output line %q: %v", line, err)
		}
		var id uint64
		_ = json.Unmarshal(m["id"], &id)
		got[id] = m
	}
	return got
}

func TestServerCallsBindings(t *testing.T) {
	s := config.NewMemoryStore()
	got := roundTrip(t, s, `{"id":1,"method":"AddRepo","args":["api","git@example.com:api.git"]}`)
	if e := got[1]["error"]; e != nil {
		t.Fatalf("AddRepo error: %s", e)
	}

	got = roundTrip(t, s, `{"id":2,"method":"ListRepos","args":[]}`)
	var repos map[string]RepoDTO
	if err := json.Unmarshal(got[2]["result"], &repos); err != nil {
		t.Fatalf("decode result: %v", err)
	}
	if repos["api"].URL != "git@example.com:api.git" {
		t.Fatalf("ListRepos = %+v", repos)
	}
}

// A nil slice from Go must reach the UI as [] rather than null: the
// Settings page reads .length on ListOrphanWorkspaces' result, and a
// fresh install (no root set) returns nil.
func TestServerSendsEmptyListsNotNull(t *testing.T) {
	got := roundTrip(t, config.NewMemoryStore(),
		`{"id":1,"method":"ListOrphanWorkspaces","args":[]}`)
	if r := string(got[1]["result"]); r != "[]" {
		t.Fatalf("ListOrphanWorkspaces result = %s, want []", r)
	}

	type inner struct{ Tags []string }
	type outer struct {
		Items []inner
		ByKey map[string][]string
		Ptr   *inner
	}
	in := outer{Items: []inner{{}}, ByKey: map[string][]string{"a": nil}}
	buf, err := json.Marshal(emptyNils(reflect.ValueOf(in)).Interface())
	if err != nil {
		t.Fatal(err)
	}
	want := `{"Items":[{"Tags":[]}],"ByKey":{"a":[]},"Ptr":null}`
	if string(buf) != want {
		t.Fatalf("emptyNils = %s, want %s", buf, want)
	}
}

func TestServerErrors(t *testing.T) {
	cases := map[string]string{
		`{"id":1,"method":"NoSuchMethod","args":[]}`:      "unknown method",
		`{"id":1,"method":"AddRepo","args":["only-one"]}`: "want 2 args",
		`{"id":1,"method":"AddRepo","args":[1,2]}`:        "arg 0",
		`{"id":1,"method":"AddRepo","args":["","url"]}`:   "must not be empty",
	}
	for req, want := range cases {
		got := roundTrip(t, config.NewMemoryStore(), req)
		var msg string
		_ = json.Unmarshal(got[1]["error"], &msg)
		if !strings.Contains(msg, want) {
			t.Errorf("%s: error %q, want it to contain %q", req, msg, want)
		}
	}
}

func TestServerEmitsEvents(t *testing.T) {
	var out bytes.Buffer
	srv := NewServer(context.Background(), config.NewMemoryStore(), &out)
	srv.Emit("workspace:progress", "cloning api")

	var ev rpcEvent
	if err := json.Unmarshal(out.Bytes(), &ev); err != nil {
		t.Fatalf("decode event: %v", err)
	}
	if ev.Event != "workspace:progress" || ev.Data != "cloning api" {
		t.Fatalf("event = %+v", ev)
	}
}

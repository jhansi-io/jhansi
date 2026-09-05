package main

import (
	"encoding/json"
	"github.com/jhansi-io/jhansi/internal/isolation"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestNewServerCreatesDataDir asserts construction is self-sufficient: it
// creates the data dir 0o700 and opens the sink inside it, so a first run
// against an empty path works with no setup
func TestNewServerCreatesDataDir(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "data")

	if _, err := newServer(Config{Addr: ":0", DataDir: dir}, &isolation.StubEngine{}); err != nil {
		t.Fatalf("newServer: %v", err)
	}

	fi, err := os.Stat(dir)
	if err != nil {
		t.Fatalf("stat data dir: %v", err)
	}
	if got := fi.Mode().Perm(); got != 0o700 {
		t.Errorf("data dir mode = %o, want 700", got)
	}
	if _, err := os.Stat(filepath.Join(dir, "events.jsonl")); err != nil {
		t.Errorf("sink file: %v", err)
	}
}

// TestServerEndToEnd drives the wired handler through create → execx and
// asserts the run's full event spine landed on disk. The only test in the
// tree that goes through the real FileSink to a real file — every other
// sink in tests is in-memory, so this is the first proof the evidence
// spine survives the wiring.
func TestServerEndToEnd(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "data")
	srv, err := newServer(Config{Addr: ":0", DataDir: dir}, &isolation.StubEngine{})
	if err != nil {
		t.Fatalf("newServer: %v", err)
	}

	req := httptest.NewRequest("POST", "/v1/sandboxes", nil)
	rec := httptest.NewRecorder()
	srv.Handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create status = %d, want 201", rec.Code)
	}

	var created struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&created); err != nil {
		t.Fatalf("decode create: %v", err)
	}

	body := strings.NewReader(`{"command": "echo hi"}`)
	req = httptest.NewRequest("POST", "/v1/sandboxes/"+created.ID+"/exec", body)
	rec = httptest.NewRecorder()
	srv.Handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("exec status = %d, want 200", rec.Code)
	}

	data, err := os.ReadFile(filepath.Join(dir, "events.jsonl"))
	if err != nil {
		t.Fatalf("read events: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) != 8 {
		t.Fatalf("events = %d, want 8:\n%s", len(lines), data)
	}
}

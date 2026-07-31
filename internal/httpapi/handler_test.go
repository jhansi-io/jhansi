package httpapi

import (
	"context"
	"encoding/json"
	"github.com/jhansi-io/jhansi/internal/evidence"
	"github.com/jhansi-io/jhansi/internal/isolation"
	"github.com/jhansi-io/jhansi/internal/registry"
	"github.com/jhansi-io/jhansi/internal/service"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
)

func TestCreateSandbox(t *testing.T) {
	sink, err := evidence.NewFileSink(filepath.Join(t.TempDir(), "events.jsonl"))
	if err != nil {
		t.Fatalf("new sink: %v", err)
	}
	h := New(service.New(registry.New(), sink, &isolation.StubEngine{}))
	req := httptest.NewRequest("POST", "/v1/sandboxes", nil)
	rec := httptest.NewRecorder()
	h.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status: got %d, want %d", rec.Code, http.StatusCreated)
	}
	var resp sandboxResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.ID == "" {
		t.Error("id: got empty, want non-empty")
	}
	if resp.Status != "READY" {
		t.Errorf("status: got %q, want %q", resp.Status, "READY")
	}
}

func TestGetSandbox(t *testing.T) {
	sink, err := evidence.NewFileSink(filepath.Join(t.TempDir(), "events.jsonl"))
	if err != nil {
		t.Fatalf("new sink: %v", err)
	}
	svc := service.New(registry.New(), sink, &isolation.StubEngine{})
	sb, err := svc.CreateSandbox()
	if err != nil {
		t.Fatalf("seed: %v", err)
	}
	h := New(svc)

	req := httptest.NewRequest("GET", "/v1/sandboxes/"+sb.ID, nil)
	rec := httptest.NewRecorder()
	h.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status: got %d, want %d", rec.Code, http.StatusOK)
	}
	var resp sandboxResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.ID != sb.ID {
		t.Errorf("id: got %q, want %q", resp.ID, sb.ID)
	}
}

func TestSandboxNotFound(t *testing.T) {
	sink, err := evidence.NewFileSink(filepath.Join(t.TempDir(), "events.jsonl"))
	if err != nil {
		t.Fatalf("new sink: %v", err)
	}
	h := New(service.New(registry.New(), sink, &isolation.StubEngine{}))
	req := httptest.NewRequest("GET", "/v1/sandboxes/sb_missing", nil)
	rec := httptest.NewRecorder()
	h.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status: got %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestListSandboxesEmpty(t *testing.T) {
	sink, err := evidence.NewFileSink(filepath.Join(t.TempDir(), "events.jsonl"))
	if err != nil {
		t.Fatalf("new sink: %v", err)
	}
	h := New(service.New(registry.New(), sink, &isolation.StubEngine{}))

	req := httptest.NewRequest("GET", "/v1/sandboxes", nil)
	rec := httptest.NewRecorder()
	h.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status: got %d, want %d", rec.Code, http.StatusOK)
	}
	if got := rec.Body.String(); got != "[]\n" {
		t.Errorf("body: got %q, want %q", got, "[]\n")
	}
}

func TestListSandboxes(t *testing.T) {
	sink, err := evidence.NewFileSink(filepath.Join(t.TempDir(), "events.jsonl"))

	if err != nil {
		t.Fatalf("new sink: %v", err)
	}
	svc := service.New(registry.New(), sink, &isolation.StubEngine{})
	a, err := svc.CreateSandbox()
	if err != nil {
		t.Fatalf("seed a: %v", err)
	}
	b, err := svc.CreateSandbox()
	if err != nil {
		t.Fatalf("seed b: %v", err)
	}
	h := New(svc)

	req := httptest.NewRequest("GET", "/v1/sandboxes", nil)
	rec := httptest.NewRecorder()
	h.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status: got %d, want %d", rec.Code, http.StatusOK)
	}
	var resp []sandboxResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(resp) != 2 {
		t.Fatalf("count: got %d, want 2", len(resp))
	}
	got := map[string]bool{resp[0].ID: true, resp[1].ID: true}
	if !got[a.ID] || !got[b.ID] {
		t.Errorf("ids: got %v, want %s and %s", got, a.ID, b.ID)
	}
}

func TestDeleteSandbox(t *testing.T) {
	sink, err := evidence.NewFileSink(filepath.Join(t.TempDir(), "events.jsonl"))
	if err != nil {
		t.Fatalf("new sink: %v", err)
	}
	svc := service.New(registry.New(), sink, &isolation.StubEngine{})
	sb, err := svc.CreateSandbox()
	if err != nil {
		t.Fatalf("seed: %v", err)
	}
	h := New(svc)

	req := httptest.NewRequest("DELETE", "/v1/sandboxes/"+sb.ID, nil)
	rec := httptest.NewRecorder()
	h.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status: got %d, want %d", rec.Code, http.StatusNoContent)
	}
	if rec.Body.Len() != 0 {
		t.Errorf("body: got %q, want empty", rec.Body.String())
	}
}

func TestDeleteSandboxNotFound(t *testing.T) {
	sink, err := evidence.NewFileSink(filepath.Join(t.TempDir(), "events.jsonl"))
	if err != nil {
		t.Fatalf("new sink: %v", err)
	}
	h := New(service.New(registry.New(), sink, &isolation.StubEngine{}))

	req := httptest.NewRequest("DELETE", "/v1/sandboxes/sb_missing", nil)
	rec := httptest.NewRecorder()

	h.Routes().ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status: got %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestExec(t *testing.T) {
	sink, err := evidence.NewFileSink(filepath.Join(t.TempDir(), "events.jsonl"))
	if err != nil {
		t.Fatalf("new sink: %v", err)
	}
	svc := service.New(registry.New(), sink, &isolation.StubEngine{})

	sb, err := svc.CreateSandbox()
	if err != nil {
		t.Fatalf("CreateSandbox: %v", err)
	}
	h := New(svc)

	body := strings.NewReader(`{"command": "echo hello"}`)
	req := httptest.NewRequest("POST", "/v1/sandboxes/"+sb.ID+"/exec", body)
	rec := httptest.NewRecorder()

	h.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}

	var resp execResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !strings.HasPrefix(resp.RunID, "run_") {
		t.Errorf("run_id = %q, want run_ prefix", resp.RunID)
	}
	if resp.Status != "SUCCEEDED" {
		t.Errorf("status = %q, want SUCCEEDED", resp.Status)
	}
	if resp.ExitCode != 0 {
		t.Errorf("exit_code = %d, want 0", resp.ExitCode)
	}
	if resp.Stdout != "echo hello" {
		t.Errorf("stdout = %q, want the command echoed", resp.Stdout)
	}
}

func TestExecNonZeroExitIs200(t *testing.T) {
	sink, err := evidence.NewFileSink(filepath.Join(t.TempDir(), "events.jsonl"))
	if err != nil {
		t.Fatalf("new sink: %v", err)
	}
	engine := &isolation.StubEngine{
		ExecFunc: func(ctx context.Context, sandboxID, command string) (isolation.ExecResult, error) {
			return isolation.ExecResult{ExitCode: 1, Stderr: "boom"}, nil
		},
	}
	svc := service.New(registry.New(), sink, engine)

	sb, err := svc.CreateSandbox()
	if err != nil {
		t.Fatalf("CreateSandbox: %v", err)
	}
	h := New(svc)

	body := strings.NewReader(`{"command": "false"}`)
	req := httptest.NewRequest("POST", "/v1/sandboxes/"+sb.ID+"/exec", body)
	rec := httptest.NewRecorder()

	h.Routes().ServeHTTP(rec, req)

	// The command failed; the HTTP call did not. This is the whole 200-not-500 decision.
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 — a failed command is a successful call", rec.Code)
	}

	var resp execResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Status != "FAILED" {
		t.Errorf("status = %q, want FAILED", resp.Status)
	}
	if resp.ExitCode != 1 {
		t.Errorf("exit_code = %d, want 1", resp.ExitCode)
	}
}

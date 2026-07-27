package httpapi

import (
	"encoding/json"
	"github.com/jhansi-io/jhansi/internal/evidence"
	"github.com/jhansi-io/jhansi/internal/registry"
	"github.com/jhansi-io/jhansi/internal/service"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
)

func TestCreateSandbox(t *testing.T) {
	sink, err := evidence.NewFileSink(filepath.Join(t.TempDir(), "events.jsonl"))
	if err != nil {
		t.Fatalf("new sink: %v", err)
	}
	h := New(service.New(registry.New(), sink))
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
	if resp.Status != "CREATING" {
		t.Errorf("status: got %q, want %q", resp.Status, "CREATING")
	}
}

func TestGetSandbox(t *testing.T) {
	sink, err := evidence.NewFileSink(filepath.Join(t.TempDir(), "events.jsonl"))
	if err != nil {
		t.Fatalf("new sink: %v", err)
	}
	svc := service.New(registry.New(), sink)
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
	h := New(service.New(registry.New(), sink))
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
	h := New(service.New(registry.New(), sink))

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
	svc := service.New(registry.New(), sink)
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

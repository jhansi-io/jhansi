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

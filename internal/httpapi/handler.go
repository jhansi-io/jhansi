package httpapi

import (
	"encoding/json"
	"errors"
	"github.com/jhansi-io/jhansi/internal/registry"
	"github.com/jhansi-io/jhansi/internal/service"
	"net/http"
)

// Handler serves the jhansi HTTP API over an Execution Service.
type Handler struct {
	svc *service.ExecutionService
}

// SandboxResponse is the wire shape for a sandbox. Kept separate from
// the domain aggregate so the wire contract stays stable (ADR-009).
type sandboxResponse struct {
	ID     string `json:"id"`
	Status string `json:"status"`
}

// New constructs a Handler over an ExecutionService
func New(svc *service.ExecutionService) *Handler {
	return &Handler{svc: svc}
}

// CreateSandbox handles POST /v1/sandboxes. On success it writes 201
// with the sandbox JSON; any error is a 500 (ADR-009 — no mapping layer
// earned yet, every reachable failure here is internal).
func (h *Handler) CreateSandbox(w http.ResponseWriter, r *http.Request) {
	sb, err := h.svc.CreateSandbox()
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	resp := sandboxResponse{ID: sb.ID, Status: string(sb.Status)}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(resp)
}

// GetSandbox handles GET /v1/sandboxes/{id}. Unknown id → 404, any
// other error → 500 (ADR-010) — inline, no mapping layer yet.
func (h *Handler) GetSandbox(w http.ResponseWriter, r *http.Request) {
	sb, err := h.svc.GetSandbox(r.PathValue("id"))
	if err != nil {
		if errors.Is(err, registry.ErrNotFound) {
			http.Error(w, "sandbox not found", http.StatusNotFound)
			return
		}
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	resp := sandboxResponse{ID: sb.ID, Status: string(sb.Status)}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}

// ListSandboxes handles GET /v1/sandboxes. Always 200 with a JSON
// array; there's no failable path (ADR-009 patterns, no new decision).
func (h *Handler) ListSandboxes(w http.ResponseWriter, r *http.Request) {
	sbs := h.svc.ListSandboxes()

	resp := make([]sandboxResponse, 0, len(sbs))
	for _, sb := range sbs {
		resp = append(resp, sandboxResponse{ID: sb.ID, Status: string(sb.Status)})
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}

// Routes returns the mux with API's routes registered. Server
// bootstraps (main, address, timeouts) stays deferred (ADR-009).
func (h *Handler) Routes() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /v1/sandboxes", h.CreateSandbox)
	mux.HandleFunc("GET /v1/sandboxes/{id}", h.GetSandbox)
	mux.HandleFunc("GET /v1/sandboxes", h.ListSandboxes)
	return mux
}

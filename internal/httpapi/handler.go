package httpapi

import (
	"encoding/json"
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

// Routes returns the mux with API's routes registered. Server
// bootstraps (main, address, timeouts) stays deferred (ADR-009).
func (h *Handler) Routes() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /v1/sandboxes", h.CreateSandbox)
	return mux
}

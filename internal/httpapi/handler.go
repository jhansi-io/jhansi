package httpapi

import (
	"encoding/json"
	"errors"
	"github.com/jhansi-io/jhansi/internal/domain"
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

// execRequest is the wire shape for an exec call. One field: the seam
// takes a bare command string and no language field (ADR-012), and the
// DTO mirrors it exactly.
type execRequest struct {
	Command string `json:"command"`
}

// execResponse is the wire shape for an exec result — assembled from the
// engine's ExecResult plus the run id, never from a lookup (ADR-015).
// Deliberately not isolation.ExecResult: that is the seam's contract, it
// carries TimedOut which never goes on the wire, and it has never heard
// of a run.
type execResponse struct {
	RunID    string `json:"run_id"`
	Status   string `json:"status"`
	ExitCode int    `json:"exit_code"`
	Stdout   string `json:"stdout"`
	Stderr   string `json:"stderr"`
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
		code := statusFor(err)
		http.Error(w, http.StatusText(code), code)
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
		code := statusFor(err)
		http.Error(w, http.StatusText(code), code)
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

// DeleteSandbox handles DELETE /v1/sandboxes/{id}. Idempotent: a
// successful or already-deleted sandbox → 204, unknown id → 404, any
// other error → 500 (ADR-011) — inline, no mapping layer yet.
func (h *Handler) DeleteSandbox(w http.ResponseWriter, r *http.Request) {
	err := h.svc.DeleteSandbox(r.PathValue("id"))
	if err != nil {
		code := statusFor(err)
		http.Error(w, http.StatusText(code), code)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// Exec handles POST /v1/sandboxes/{id}/exec. Unknown sandbox → 404;
// infra fault → 500; a completed run — success, non-zero exit, or timeout
// — is 200 with the real terminal in Status (ADR-017). Mapping stays
// inline; the busy-sandbox 409 and a mapping layer are ADR-018.
func (h *Handler) Exec(w http.ResponseWriter, r *http.Request) {
	var req execRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	run, result, err := h.svc.Exec(r.Context(), r.PathValue("id"), req.Command)
	if err != nil {
		code := statusFor(err)
		http.Error(w, http.StatusText(code), code)
		return
	}
	resp := execResponse{
		RunID:    run.ID,
		Status:   string(run.Status),
		ExitCode: result.ExitCode,
		Stdout:   result.Stdout,
		Stderr:   result.Stderr,
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
	mux.HandleFunc("DELETE /v1/sandboxes/{id}", h.DeleteSandbox)
	mux.HandleFunc("POST /v1/sandboxes/{id}/exec", h.Exec)
	return mux
}

// statusFor maps a service error to an HTTP status. Unknown id → 404,
// busy sandbox → 409, anything unclassified → 500 (deny-by-default: an
// unmapped error is a server fault until proven otherwise). See ADR-018.
func statusFor(err error) int {
	switch {
	case errors.Is(err, registry.ErrNotFound):
		return http.StatusNotFound
	case errors.Is(err, domain.ErrSandboxBusy):
		return http.StatusConflict
	default:
		return http.StatusInternalServerError
	}
}

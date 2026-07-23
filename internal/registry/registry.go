package registry

import (
	"errors"
	"fmt"
	"github.com/jhansi-io/jhansi/internal/domain"
	"sync"
)

var ErrAlreadyExists = errors.New("sandbox already exists")
var ErrNotFound = errors.New("sandbox not found")

// Registry holds sandboxes in memory for the life of the process.
// Persistence is a Tier-1 re-add behind this same surface (ADR-005).
type Registry struct {
	mu        sync.RWMutex
	sandboxes map[string]*domain.Sandbox
}

func New() *Registry {
	return &Registry{
		sandboxes: make(map[string]*domain.Sandbox),
	}
}

// Add stores a sandbox. It refuses to overwrite an existing id, so a
// caller bug cannot silently clobber a live sandbox and lose its
// unflushed events (ADR-005).
func (r *Registry) Add(s *domain.Sandbox) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.sandboxes[s.ID]; ok {
		return fmt.Errorf("%s: %w", s.ID, ErrAlreadyExists)
	}
	r.sandboxes[s.ID] = s
	return nil
}

// Get returns the sandbox stored under id, or ErrNotFound.
func (r *Registry) Get(id string) (*domain.Sandbox, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	s, ok := r.sandboxes[id]
	if !ok {
		return nil, fmt.Errorf("%s: %w", id, ErrNotFound)
	}
	return s, nil
}

// List returns every stored sandbox, in unspecified order.
func (r *Registry) List() []*domain.Sandbox {
	r.mu.RLock()
	defer r.mu.RUnlock()

	out := make([]*domain.Sandbox, 0, len(r.sandboxes))
	for _, s := range r.sandboxes {
		out = append(out, s)
	}
	return out
}

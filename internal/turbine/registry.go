package turbine

import (
	"fmt"
	"sync"

	"windfarm-turbine-monitor-service/internal/platform"
)

// Registry is a thread-safe in-memory turbine fleet store.
type Registry struct {
	mu       sync.RWMutex
	turbines map[string]Turbine
	order    []string
}

func NewRegistry(seed []Turbine) *Registry {
	r := &Registry{turbines: make(map[string]Turbine, len(seed))}
	for _, t := range seed {
		r.turbines[t.ID] = t
		r.order = append(r.order, t.ID)
	}
	return r
}

func (r *Registry) Put(t Turbine) error {
	if t.ID == "" {
		return fmt.Errorf("turbine id required: %w", platform.ErrInvalid)
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.turbines[t.ID]; !exists {
		r.order = append(r.order, t.ID)
	}
	r.turbines[t.ID] = t
	return nil
}

func (r *Registry) Get(id string) (Turbine, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	t, ok := r.turbines[id]
	if !ok {
		return Turbine{}, fmt.Errorf("turbine %s: %w", id, platform.ErrNotFound)
	}
	return t, nil
}

// List returns the fleet in registration order, reflecting the current state.
func (r *Registry) List() []Turbine {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]Turbine, 0, len(r.order))
	for _, id := range r.order {
		out = append(out, r.turbines[id])
	}
	return out
}

func (r *Registry) ListActive() []Turbine {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]Turbine, 0, len(r.order))
	for _, id := range r.order {
		if t := r.turbines[id]; t.Active() {
			out = append(out, t)
		}
	}
	return out
}

func (r *Registry) UpdateState(id string, state TurbineState) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	t, ok := r.turbines[id]
	if !ok {
		return fmt.Errorf("turbine %s: %w", id, platform.ErrNotFound)
	}
	t.State = state
	r.turbines[id] = t
	return nil
}

func (r *Registry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.order)
}

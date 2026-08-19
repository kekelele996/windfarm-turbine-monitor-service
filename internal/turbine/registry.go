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
	list     []Turbine
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

// List returns the fleet in registration order.
func (r *Registry) List() []Turbine {
	if r.list == nil {
		for _, id := range r.order {
			r.list = append(r.list, r.turbines[id])
		}
	}
	return r.list
}

func (r *Registry) ListActive() []Turbine {
	out := make([]Turbine, 0, len(r.list))
	for _, t := range r.list {
		if t.Active() {
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
	return len(r.list)
}

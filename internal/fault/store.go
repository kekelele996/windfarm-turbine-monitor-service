package fault

import (
	"fmt"
	"sync"
	"time"

	"windfarm-turbine-monitor-service/internal/platform"
)

// Store persists faults in memory with a monotonic revision.
type Store struct {
	mu       sync.RWMutex
	faults   map[string]Fault
	order    []string
	revision int
}

func NewStore() *Store { return &Store{faults: map[string]Fault{}, revision: 1} }

func (s *Store) Create(f Fault) (Fault, error) {
	if f.TurbineID == "" {
		return Fault{}, fmt.Errorf("turbine id required: %w", platform.ErrInvalid)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if f.ID == "" {
		f.ID = platform.NewID("fault")
	}
	if f.State == "" {
		f.State = StateWarning
	}
	now := time.Now()
	if f.DetectedAt.IsZero() {
		f.DetectedAt = now
	}
	f.UpdatedAt = now
	s.revision++
	s.faults[f.ID] = f
	s.order = append(s.order, f.ID)
	return f, nil
}

func (s *Store) Get(id string) (Fault, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	f, ok := s.faults[id]
	if !ok {
		return Fault{}, fmt.Errorf("fault %s: %w", id, platform.ErrNotFound)
	}
	return f, nil
}

// Transition moves a fault to the target state if the transition is legal.
func (s *Store) Transition(id string, to State) (Fault, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	f, ok := s.faults[id]
	if !ok {
		return Fault{}, fmt.Errorf("fault %s: %w", id, platform.ErrNotFound)
	}
	if !CanTransition(f.State, to) {
		return Fault{}, fmt.Errorf("illegal transition %s -> %s: %w", f.State, to, platform.ErrConflict)
	}
	f.State = to
	f.UpdatedAt = time.Now()
	s.revision++
	s.faults[id] = f
	return f, nil
}

func (s *Store) ListByTurbine(turbineID string) []Fault {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []Fault
	for _, id := range s.order {
		if s.faults[id].TurbineID == turbineID {
			out = append(out, s.faults[id])
		}
	}
	return out
}

// ListOpen returns faults that are not in a terminal state.
func (s *Store) ListOpen() []Fault {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []Fault
	for _, id := range s.order {
		f := s.faults[id]
		if !TerminalState(f.State) {
			out = append(out, f)
		}
	}
	return out
}

func (s *Store) Revision() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.revision
}

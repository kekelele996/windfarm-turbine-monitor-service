package fault

import (
	"errors"
	"fmt"

	"windfarm-turbine-monitor-service/internal/platform"
)

// Service coordinates fault lifecycle around a Store.
type Service struct {
	store *Store
}

func NewService(store *Store) *Service { return &Service{store: store} }

// Raise creates a fault if an equivalent open fault does not already exist for
// the same turbine and code.
func (svc *Service) Raise(turbineID, code, severity string) (Fault, error) {
	for _, f := range svc.store.ListOpen() {
		if f.TurbineID == turbineID && f.Code == code {
			return f, nil
		}
	}
	return svc.store.Create(Fault{TurbineID: turbineID, Code: code, Severity: severity, State: StateWarning})
}

// Acknowledge moves a fault toward maintenance.
func (svc *Service) Acknowledge(id string) (Fault, error) {
	f, err := svc.store.Get(id)
	if err != nil {
		if errors.Is(err, platform.ErrNotFound) {
			return Fault{}, err
		}
		return Fault{}, fmt.Errorf("lookup fault: %w", err)
	}
	if f.State == StateNormal {
		return f, nil
	}
	return svc.store.Transition(id, StateMaintenance)
}

// Resolve marks a fault as back to normal after maintenance.
func (svc *Service) Resolve(id string) (Fault, error) {
	f, err := svc.store.Get(id)
	if err != nil {
		if errors.Is(err, platform.ErrNotFound) {
			return Fault{}, err
		}
		return Fault{}, fmt.Errorf("lookup fault: %w", err)
	}
	if !CanTransition(f.State, StateNormal) {
		return Fault{}, fmt.Errorf("cannot resolve fault in state %s: %w", f.State, platform.ErrConflict)
	}
	return svc.store.Transition(id, StateNormal)
}

// Open returns all faults that are not yet resolved to a terminal state.
func (svc *Service) Open() []Fault { return svc.store.ListOpen() }

// Get returns a fault by id.
func (svc *Service) Get(id string) (Fault, error) { return svc.store.Get(id) }

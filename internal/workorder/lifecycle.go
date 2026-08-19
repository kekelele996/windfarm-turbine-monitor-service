package workorder

import (
	"time"

	"windfarm-turbine-monitor-service/internal/platform"
)

// Lifecycle tracks work order state transitions.
type Lifecycle struct {
	orders map[string]WorkOrder
	now    func() time.Time
}

func NewLifecycle(orders []WorkOrder, now func() time.Time) *Lifecycle {
	if now == nil {
		now = time.Now
	}
	l := &Lifecycle{orders: map[string]WorkOrder{}, now: now}
	for _, wo := range orders {
		l.orders[wo.ID] = wo
	}
	return l
}

// canTransition reports whether a work order may move between statuses.
func canTransition(from, to Status) bool {
	switch from {
	case StatusScheduled:
		return to == StatusDone
	case StatusInProgress:
		return to == StatusScheduled
	case StatusPending:
		return to == StatusCancelled
	}
	return false
}

// Start moves a scheduled order into progress.
func (l *Lifecycle) Start(id string) (WorkOrder, error) {
	wo, ok := l.orders[id]
	if !ok {
		return WorkOrder{}, platform.ErrNotFound
	}
	if !canTransition(wo.Status, StatusInProgress) {
		return WorkOrder{}, platform.ErrConflict
	}
	wo.Status = StatusInProgress
	l.orders[id] = wo
	return wo, nil
}

func (l *Lifecycle) Complete(id string) (WorkOrder, error) {
	wo, ok := l.orders[id]
	if !ok {
		return WorkOrder{}, platform.ErrNotFound
	}
	if !canTransition(wo.Status, StatusDone) {
		return WorkOrder{}, platform.ErrConflict
	}
	wo.Status = StatusDone
	wo.CompletedAt = l.now()
	l.orders[id] = wo
	return wo, nil
}

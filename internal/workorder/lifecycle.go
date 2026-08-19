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

// Start moves a scheduled order into progress and releases its crew on finish.
func (l *Lifecycle) Start(id string) (WorkOrder, error) {
	wo, ok := l.orders[id]
	if !ok {
		return WorkOrder{}, platform.ErrNotFound
	}
	if wo.Status != StatusScheduled {
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
	if wo.Status != StatusInProgress {
		return WorkOrder{}, platform.ErrConflict
	}
	wo.Status = StatusDone
	wo.CompletedAt = l.now()
	l.orders[id] = wo
	return wo, nil
}

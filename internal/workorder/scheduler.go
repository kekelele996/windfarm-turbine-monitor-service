package workorder

import (
	"fmt"
	"time"

	"windfarm-turbine-monitor-service/internal/platform"
)

// Scheduler creates work orders and assigns crews by availability.
type Scheduler struct {
	orders map[string]WorkOrder
	crews  *CrewPool
	now    func() time.Time
}

func NewScheduler(crews *CrewPool, now func() time.Time) *Scheduler {
	if now == nil {
		now = time.Now
	}
	return &Scheduler{orders: map[string]WorkOrder{}, crews: crews, now: now}
}

// Schedule creates a pending work order for a turbine.
func (s *Scheduler) Schedule(turbineID, title string, priority int) WorkOrder {
	wo := WorkOrder{
		ID:        platform.NewID("wo"),
		TurbineID: turbineID,
		Title:     title,
		Priority:  priority,
		Status:    StatusPending,
		CreatedAt: s.now(),
	}
	s.orders[wo.ID] = wo
	return wo
}

// Assign tries to attach a free crew to a pending order.
func (s *Scheduler) Assign(orderID string, preferred string) (WorkOrder, error) {
	wo, ok := s.orders[orderID]
	if !ok {
		return WorkOrder{}, fmt.Errorf("order %s: %w", orderID, platform.ErrNotFound)
	}
	if wo.Status != StatusPending {
		return WorkOrder{}, fmt.Errorf("order not pending: %w", platform.ErrConflict)
	}
	crew, ok := s.crews.Assign(preferred)
	if !ok {
		return WorkOrder{}, fmt.Errorf("no crew available: %w", platform.ErrUnavailable)
	}
	wo.CrewID = crew
	wo.Status = StatusScheduled
	wo.PlannedAt = s.now().Add(24 * time.Hour)
	s.orders[orderID] = wo
	return wo, nil
}

func (s *Scheduler) List() []WorkOrder {
	out := make([]WorkOrder, 0, len(s.orders))
	for _, wo := range s.orders {
		out = append(out, wo)
	}
	return out
}

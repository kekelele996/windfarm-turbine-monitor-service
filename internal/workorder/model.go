package workorder

import "time"

type Status string

const (
	StatusPending    Status = "pending"
	StatusScheduled  Status = "scheduled"
	StatusInProgress Status = "in_progress"
	StatusDone       Status = "done"
	StatusCancelled  Status = "cancelled"
)

// WorkOrder schedules maintenance for a turbine.
type WorkOrder struct {
	ID          string    `json:"id"`
	TurbineID   string    `json:"turbine_id"`
	Title       string    `json:"title"`
	Priority    int       `json:"priority"`
	CrewID      string    `json:"crew_id"`
	Status      Status    `json:"status"`
	PlannedAt   time.Time `json:"planned_at"`
	CreatedAt   time.Time `json:"created_at"`
	CompletedAt time.Time `json:"completed_at"`
}

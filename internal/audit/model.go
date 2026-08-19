package audit

import "time"

type EventType string

const (
	EventIngest    EventType = "ingest"
	EventAlarm     EventType = "alarm"
	EventFault     EventType = "fault"
	EventWorkOrder EventType = "work_order"
)

// Event is an immutable audit record.
type Event struct {
	ID        string    `json:"id"`
	Type      EventType `json:"type"`
	TurbineID string    `json:"turbine_id"`
	Payload   string    `json:"payload"`
	At        time.Time `json:"at"`
}

package alarm

import "time"

type Status string

const (
	StatusOpen       Status = "open"
	StatusAck        Status = "acknowledged"
	StatusResolved   Status = "resolved"
	StatusSuppressed Status = "suppressed"
)

// Alarm is a notification derived from a rule evaluation.
type Alarm struct {
	ID         string    `json:"id"`
	TurbineID  string    `json:"turbine_id"`
	RuleID     string    `json:"rule_id"`
	Metric     string    `json:"metric"`
	Severity   string    `json:"severity"`
	Value      float64   `json:"value"`
	Threshold  float64   `json:"threshold"`
	Status     Status    `json:"status"`
	OccurredAt time.Time `json:"occurred_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// Key is the dedup identity for an alarm.
func (a Alarm) Key() string { return a.TurbineID + ":" + a.RuleID }

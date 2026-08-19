package fault

import "time"

type State string

const (
	StateNormal      State = "normal"
	StateWarning     State = "warning"
	StateFault       State = "fault"
	StateMaintenance State = "maintenance"
	StateRetrying    State = "retrying"
)

// Fault is one diagnosed issue attached to a turbine.
type Fault struct {
	ID         string    `json:"id"`
	TurbineID  string    `json:"turbine_id"`
	Code       string    `json:"code"`
	State      State     `json:"state"`
	Severity   string    `json:"severity"`
	DetectedAt time.Time `json:"detected_at"`
	UpdatedAt  time.Time `json:"updated_at"`
	Notes      string    `json:"notes"`
}

package energy

import (
	"time"

	"windfarm-turbine-monitor-service/internal/grid"
)

// ExportPlan reconciles curtailment against metered energy.
type ExportPlan struct {
	Site         string
	CurtailedKWh float64
	ExportedKWh  float64
}

// Reconcile applies an active curtailment limit to metered energy for a site.
// When a curtailment is active the exportable energy is capped; the difference
// is reported as curtailed.
func Reconcile(site string, meteredKWh float64, sched *grid.Schedule, at time.Time, intervalHours float64) ExportPlan {
	plan := ExportPlan{Site: site, ExportedKWh: meteredKWh}
	if intervalHours <= 0 {
		intervalHours = 1
	}
	limitKW, active := sched.ActiveFor(site, at)
	if !active {
		return plan
	}
	limitKWh := limitKW * intervalHours
	if meteredKWh > limitKWh {
		plan.CurtailedKWh = meteredKWh - limitKWh
		plan.ExportedKWh = limitKWh
	}
	return plan
}

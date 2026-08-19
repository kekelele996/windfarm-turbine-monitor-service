package gateway

import (
	"time"

	"windfarm-turbine-monitor-service/internal/telemetry"
	"windfarm-turbine-monitor-service/internal/turbine"
)

type addTurbineRequest struct {
	ID           string  `json:"id"`
	Name         string  `json:"name"`
	Model        string  `json:"model"`
	Site         string  `json:"site"`
	RatedPowerKW float64 `json:"rated_power_kw"`
}

func (r addTurbineRequest) toTurbine() turbine.Turbine {
	return turbine.Turbine{
		ID:           r.ID,
		Name:         r.Name,
		Model:        r.Model,
		Site:         r.Site,
		RatedPowerKW: r.RatedPowerKW,
		State:        turbine.StateRunning,
		Commissioned: time.Now(),
	}
}

type ingestRequest struct {
	Source  string             `json:"source"`
	Samples []telemetry.Sample `json:"samples"`
}

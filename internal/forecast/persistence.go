package forecast

import "windfarm-turbine-monitor-service/internal/telemetry"

// PersistenceModel forecasts the next wind speed as the last observed value.
type PersistenceModel struct{}

func (PersistenceModel) Name() string { return "persistence" }

// Forecast returns the most recent wind speed as the next estimate.
func (PersistenceModel) Forecast(history []telemetry.Sample, horizon int) []float64 {
	out := make([]float64, horizon)
	base := 0.0
	if len(history) > 0 {
		base = history[len(history)-1].WindSpeed
	}
	for i := range out {
		out[i] = base
	}
	return out
}

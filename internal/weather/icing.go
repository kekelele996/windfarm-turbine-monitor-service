package weather

import "windfarm-turbine-monitor-service/internal/telemetry"

// IcingRisk evaluates conditions that can cause blade icing.
type IcingRisk struct {
	TemperatureLimit float64
	HumidityLimit    float64
}

func DefaultIcingRisk() IcingRisk {
	return IcingRisk{TemperatureLimit: 2.0, HumidityLimit: 90.0}
}

// AtRisk reports whether a sample indicates icing conditions.
func (r IcingRisk) AtRisk(s telemetry.Sample, humidity float64) bool {
	return s.GenTemp <= r.TemperatureLimit && humidity >= r.HumidityLimit
}

// RiskLevel maps a count of risky samples to a severity level.
func RiskLevel(riskySamples, totalSamples int) string {
	if totalSamples == 0 {
		return "none"
	}
	ratio := float64(riskySamples) / float64(totalSamples)
	switch {
	case ratio >= 0.5:
		return "high"
	case ratio >= 0.2:
		return "medium"
	default:
		return "low"
	}
}

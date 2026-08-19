package analytics

import "windfarm-turbine-monitor-service/internal/telemetry"

// Aggregate summarizes a series of samples for reporting.
type Aggregate struct {
	Count          int     `json:"count"`
	MeanWind       float64 `json:"mean_wind_mps"`
	MeanPower      float64 `json:"mean_power_kw"`
	MaxGearboxTemp float64 `json:"max_gearbox_temp_c"`
	MaxGenTemp     float64 `json:"max_gen_temp_c"`
	EnergyKWh      float64 `json:"energy_kwh"`
}

func AggregateSamples(samples []telemetry.Sample, intervalSeconds float64) Aggregate {
	a := Aggregate{Count: len(samples)}
	if len(samples) == 0 {
		return a
	}
	var wind, power float64
	for _, s := range samples {
		wind += s.WindSpeed
		power += s.PowerOutput
		if s.GearboxTemp > a.MaxGearboxTemp {
			a.MaxGearboxTemp = s.GearboxTemp
		}
		if s.GenTemp > a.MaxGenTemp {
			a.MaxGenTemp = s.GenTemp
		}
	}
	a.MeanWind = wind / float64(len(samples))
	a.MeanPower = power / float64(len(samples))
	// Average power (kW) over the interval yields kWh.
	a.EnergyKWh = a.MeanPower * (intervalSeconds / 3600.0)
	return a
}

package turbine

import "windfarm-turbine-monitor-service/internal/telemetry"

// Derating describes a temporary power cap applied to a turbine.
type Derating struct {
	TurbineID string
	Reason    string
	LimitKW   float64
}

// DeratingTable maps turbines to their current derating limits.
type DeratingTable struct {
	limits map[string]Derating
}

func NewDeratingTable() *DeratingTable { return &DeratingTable{limits: map[string]Derating{}} }

func (t *DeratingTable) Set(d Derating) { t.limits[d.TurbineID] = d }

func (t *DeratingTable) Get(turbineID string) (Derating, bool) {
	d, ok := t.limits[turbineID]
	return d, ok
}

// EffectivePower clamps a measured output to the derating limit.
func (t *DeratingTable) EffectivePower(turbineID string, measured float64) float64 {
	if d, ok := t.limits[turbineID]; ok && measured > d.LimitKW {
		return d.LimitKW
	}
	return measured
}

// Availability computes the fraction of samples where the turbine produced at
// least minPower, which approximates operational availability.
func Availability(samples []telemetry.Sample, minPower float64) float64 {
	if len(samples) == 0 {
		return 0
	}
	up := 0
	for _, s := range samples {
		if s.PowerOutput >= minPower {
			up++
		}
	}
	return float64(up) / float64(len(samples))
}

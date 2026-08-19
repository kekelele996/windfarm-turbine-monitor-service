package projection

import (
	"sync"

	"windfarm-turbine-monitor-service/internal/alarm"
)

// AlarmProjection keeps per-turbine alarm counters updated incrementally.
type AlarmProjection struct {
	mu         sync.RWMutex
	byTurbine  map[string]int
	bySeverity map[string]int
}

func NewAlarmProjection() *AlarmProjection {
	return &AlarmProjection{byTurbine: map[string]int{}, bySeverity: map[string]int{}}
}

// Apply updates counters from a single alarm.
func (p *AlarmProjection) Apply(a alarm.Alarm) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.byTurbine[a.TurbineID]++
	p.bySeverity[a.Severity]++
}

// ByTurbine returns a snapshot of per-turbine counts.
func (p *AlarmProjection) ByTurbine() map[string]int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	out := make(map[string]int, len(p.byTurbine))
	for k, v := range p.byTurbine {
		out[k] = v
	}
	return out
}

// TopSeverity returns the highest-count severity label.
func (p *AlarmProjection) TopSeverity() string {
	p.mu.RLock()
	defer p.mu.RUnlock()
	best := ""
	bestN := -1
	for s, n := range p.bySeverity {
		if n > bestN {
			best = s
			bestN = n
		}
	}
	return best
}

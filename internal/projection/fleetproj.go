package projection

import "windfarm-turbine-monitor-service/internal/turbine"

// FleetProjection maintains derived fleet views for fast reads.
type FleetProjection struct {
	activeBySite map[string]int
}

func NewFleetProjection() *FleetProjection {
	return &FleetProjection{activeBySite: map[string]int{}}
}

// Rebuild recomputes derived views from a registry snapshot.
func (p *FleetProjection) Rebuild(list []turbine.Turbine) {
	p.activeBySite = map[string]int{}
	for _, t := range list {
		if t.Active() {
			p.activeBySite[t.Site]++
		}
	}
}

// ActiveOnSite returns the active turbine count for a site.
func (p *FleetProjection) ActiveOnSite(site string) int { return p.activeBySite[site] }

// Sites returns distinct site codes with at least one active turbine.
func (p *FleetProjection) Sites() []string {
	out := make([]string, 0, len(p.activeBySite))
	for s, n := range p.activeBySite {
		if n > 0 {
			out = append(out, s)
		}
	}
	return out
}

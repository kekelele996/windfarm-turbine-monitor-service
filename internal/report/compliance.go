package report

import (
	"time"

	"windfarm-turbine-monitor-service/internal/turbine"
)

// ComplianceSection summarizes regulatory reporting obligations for a site.
type ComplianceSection struct {
	Site      string
	CheckedAt time.Time
	Warnings  []string
	Passed    bool
}

// BuildCompliance checks fleet-level obligations such as maintenance recency
// and firmware currency.
func BuildCompliance(site string, registry *turbine.Registry, now time.Time, minFirmware string) ComplianceSection {
	sec := ComplianceSection{Site: site, CheckedAt: now, Passed: true}
	for _, t := range registry.List() {
		if t.Site != site {
			continue
		}
		if t.Firmware < minFirmware {
			sec.Warnings = append(sec.Warnings, t.ID+" firmware outdated")
			sec.Passed = false
		}
		if t.State == turbine.StateMaintenance && now.Sub(t.Commissioned) < 30*24*time.Hour {
			sec.Warnings = append(sec.Warnings, t.ID+" premature maintenance")
		}
	}
	return sec
}

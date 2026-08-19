package compliance

import (
	"time"

	"windfarm-turbine-monitor-service/internal/turbine"
)

// Finding is a single compliance gap.
type Finding struct {
	Severity string
	Turbine  string
	Message  string
}

// Auditor checks fleet-level compliance against certificates.
type Auditor struct {
	certs *Store
}

func NewAuditor(certs *Store) *Auditor { return &Auditor{certs: certs} }

// Audit checks every turbine model has a valid certificate.
func (a *Auditor) Audit(registry *turbine.Registry, at time.Time) []Finding {
	var findings []Finding
	seen := map[string]bool{}
	for _, t := range registry.List() {
		if seen[t.Model] {
			continue
		}
		seen[t.Model] = true
		if !a.certs.ValidFor(t.Model, at) {
			findings = append(findings, Finding{
				Severity: "critical",
				Turbine:  t.ID,
				Message:  "model " + t.Model + " lacks a valid certificate",
			})
		}
	}
	return findings
}

// Summary reduces findings to a pass/fail status.
type Summary struct {
	Passed   bool
	Count    int
	Critical int
}

func Summarize(findings []Finding) Summary {
	s := Summary{Passed: true, Count: len(findings)}
	for _, f := range findings {
		if f.Severity == "critical" {
			s.Critical++
			s.Passed = false
		}
	}
	return s
}

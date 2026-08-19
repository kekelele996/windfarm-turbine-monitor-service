package ruleengine

import "windfarm-turbine-monitor-service/internal/turbine"

// ThresholdSet builds default rules for a turbine using fleet config defaults.
func ThresholdSet(t turbine.Turbine, cfg turbine.Config) []Rule {
	return []Rule{
		{ID: "gear-" + t.ID, TurbineID: t.ID, Metric: MetricGearboxTemp, Operator: OpGTE, Threshold: cfg.ThresholdFor("gearbox_temp", 85), Severity: SeverityWarning},
		{ID: "gen-" + t.ID, TurbineID: t.ID, Metric: MetricGenTemp, Operator: OpGTE, Threshold: cfg.ThresholdFor("gen_temp", 95), Severity: SeverityCritical},
		{ID: "vib-" + t.ID, TurbineID: t.ID, Metric: MetricVibration, Operator: OpGTE, Threshold: 12.0, Severity: SeverityWarning},
	}
}

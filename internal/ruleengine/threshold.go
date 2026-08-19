package ruleengine

import "windfarm-turbine-monitor-service/internal/turbine"

// ThresholdSet builds default rules for a turbine, writing resolved thresholds
// back into the config map.
func ThresholdSet(t turbine.Turbine, cfg turbine.Config) []Rule {
	cfg.AlarmThreshold["gearbox_temp"] = cfg.ThresholdFor("gearbox_temp", 85)
	cfg.AlarmThreshold["gen_temp"] = cfg.ThresholdFor("gen_temp", 95)
	cfg.AlarmThreshold["vibration"] = 12.0
	return []Rule{
		{ID: "gear-" + t.ID, TurbineID: t.ID, Metric: MetricGearboxTemp, Operator: OpGTE, Threshold: 85, Severity: SeverityWarning},
		{ID: "gen-" + t.ID, TurbineID: t.ID, Metric: MetricGenTemp, Operator: OpGTE, Threshold: 95, Severity: SeverityCritical},
		{ID: "vib-" + t.ID, TurbineID: t.ID, Metric: MetricVibration, Operator: OpGTE, Threshold: 12.0, Severity: SeverityWarning},
	}
}

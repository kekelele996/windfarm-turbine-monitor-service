package ruleengine

import "strings"

type Severity string

const (
	SeverityInfo     Severity = "info"
	SeverityWarning  Severity = "warning"
	SeverityCritical Severity = "critical"
)

// Metric is the name of a telemetry field a rule evaluates.
type Metric string

const (
	MetricWindSpeed   Metric = "wind_speed"
	MetricRotorRPM    Metric = "rotor_rpm"
	MetricGearboxTemp Metric = "gearbox_temp"
	MetricGenTemp     Metric = "gen_temp"
	MetricVibration   Metric = "vibration"
	MetricPowerOutput Metric = "power_output"
)

// Operator is a comparison operator.
type Operator string

const (
	OpGTE Operator = ">="
	OpLTE Operator = "<="
)

// Rule maps a telemetry metric to a severity when a threshold is crossed.
type Rule struct {
	ID        string   `json:"id"`
	TurbineID string   `json:"turbine_id"`
	Metric    Metric   `json:"metric"`
	Operator  Operator `json:"operator"`
	Threshold float64  `json:"threshold"`
	Severity  Severity `json:"severity"`
}

// AppliesTo reports whether the rule targets the given turbine (or all).
func (r Rule) AppliesTo(turbineID string) bool {
	return r.TurbineID == "" || r.TurbineID == turbineID
}

func ParseMetric(s string) (Metric, bool) {
	switch Metric(strings.ToLower(s)) {
	case MetricWindSpeed, MetricRotorRPM, MetricGearboxTemp, MetricGenTemp, MetricVibration, MetricPowerOutput:
		return Metric(strings.ToLower(s)), true
	}
	return "", false
}

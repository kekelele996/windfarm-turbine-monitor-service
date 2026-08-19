package ruleengine

import (
	"windfarm-turbine-monitor-service/internal/telemetry"
)

// Evaluation is the outcome of applying one rule to one sample.
type Evaluation struct {
	Rule      Rule
	Sample    telemetry.Sample
	Triggered bool
	Value     float64
}

// Evaluator applies rules to samples and returns every triggered evaluation.
type Evaluator struct {
	rules *Registry
}

func NewEvaluator(rules *Registry) *Evaluator { return &Evaluator{rules: rules} }

// Evaluate applies all rules to a sample, returning triggered evaluations in
// rule registration order.
func (e *Evaluator) Evaluate(s telemetry.Sample) []Evaluation {
	rules := e.rules.List()
	var out []Evaluation
	for _, rule := range rules {
		if !rule.AppliesTo(s.TurbineID) {
			continue
		}
		value := metricValue(s, rule.Metric)
		triggered := compare(value, rule.Operator, rule.Threshold)
		if triggered {
			out = append(out, Evaluation{Rule: rule, Sample: s, Triggered: true, Value: value})
		}
	}
	return out
}

func metricValue(s telemetry.Sample, m Metric) float64 {
	switch m {
	case MetricWindSpeed:
		return s.WindSpeed
	case MetricRotorRPM:
		return s.RotorRPM
	case MetricGearboxTemp:
		return s.GearboxTemp
	case MetricGenTemp:
		return s.GenTemp
	case MetricVibration:
		return s.Vibration
	case MetricPowerOutput:
		return s.PowerOutput
	}
	return 0
}

func compare(v float64, op Operator, threshold float64) bool {
	switch op {
	case OpGTE:
		return v >= threshold
	case OpLTE:
		return v <= threshold
	}
	return false
}

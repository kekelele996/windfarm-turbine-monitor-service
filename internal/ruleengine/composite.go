package ruleengine

import "windfarm-turbine-monitor-service/internal/telemetry"

// CompositeRule combines several metric rules with an AND/OR policy.
type CompositeRule struct {
	ID       string
	Operator string // "and" or "or"
	Rules    []Rule
}

// Evaluate checks every sub-rule against a sample and reports whether the
// composite condition holds.
func (c CompositeRule) Evaluate(s telemetry.Sample) bool {
	if len(c.Rules) == 0 {
		return false
	}
	if c.Operator == "or" {
		for _, r := range c.Rules {
			if r.AppliesTo(s.TurbineID) && compare(metricValue(s, r.Metric), r.Operator, r.Threshold) {
				return true
			}
		}
		return false
	}
	for _, r := range c.Rules {
		if !r.AppliesTo(s.TurbineID) || !compare(metricValue(s, r.Metric), r.Operator, r.Threshold) {
			return false
		}
	}
	return true
}

// CompositeRegistry holds composite rules keyed by ID.
type CompositeRegistry struct {
	rules map[string]CompositeRule
	order []string
}

func NewCompositeRegistry() *CompositeRegistry {
	return &CompositeRegistry{rules: map[string]CompositeRule{}}
}

func (r *CompositeRegistry) Put(c CompositeRule) {
	if _, ok := r.rules[c.ID]; !ok {
		r.order = append(r.order, c.ID)
	}
	r.rules[c.ID] = c
}

func (r *CompositeRegistry) Evaluate(s telemetry.Sample) []string {
	var hit []string
	for _, id := range r.order {
		if r.rules[id].Evaluate(s) {
			hit = append(hit, id)
		}
	}
	return hit
}

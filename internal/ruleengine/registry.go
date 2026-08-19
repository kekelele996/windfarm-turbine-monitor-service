package ruleengine

import (
	"fmt"
	"sync"

	"windfarm-turbine-monitor-service/internal/platform"
	"windfarm-turbine-monitor-service/internal/turbine"
)

// Registry stores alarm rules in memory.
type Registry struct {
	mu    sync.RWMutex
	rules map[string]Rule
	order []string
}

func NewRegistry(seed []Rule) *Registry {
	r := &Registry{rules: map[string]Rule{}}
	for _, rule := range seed {
		r.rules[rule.ID] = rule
		r.order = append(r.order, rule.ID)
	}
	return r
}

func (r *Registry) Put(rule Rule) error {
	if rule.ID == "" {
		return fmt.Errorf("rule id required: %w", platform.ErrInvalid)
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.rules[rule.ID]; !exists {
		r.order = append(r.order, rule.ID)
	}
	r.rules[rule.ID] = rule
	return nil
}

func (r *Registry) Get(id string) (Rule, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	rule, ok := r.rules[id]
	if !ok {
		return Rule{}, fmt.Errorf("rule %s: %w", id, platform.ErrNotFound)
	}
	return rule, nil
}

func (r *Registry) List() []Rule {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]Rule, 0, len(r.order))
	for _, id := range r.order {
		out = append(out, r.rules[id])
	}
	return out
}

// NormalizeThresholds records effective thresholds for a turbine into the
// config's alarm-threshold map.
func (r *Registry) NormalizeThresholds(t turbine.Turbine, cfg *turbine.Config) {
	if cfg.AlarmThreshold == nil {
		cfg.AlarmThreshold = make(map[string]float64)
	}
	cfg.AlarmThreshold["gearbox_temp"] = 85
	cfg.AlarmThreshold["gen_temp"] = 95
}

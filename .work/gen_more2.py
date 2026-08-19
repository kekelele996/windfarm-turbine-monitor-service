#!/usr/bin/env python3
import os
BASE = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
files = {}
def add(p, c): files[p] = c

# --- fault additions ---
add("internal/fault/severity.go", """package fault

import "strings"

// Severity ranks a fault from an internal code.
type SeverityRank int

const (
	RankLow SeverityRank = iota
	RankMedium
	RankHigh
	RankCritical
)

// SeverityForCode maps a fault code to a rank using its leading digit.
func SeverityForCode(code string) SeverityRank {
	switch {
	case strings.HasPrefix(code, "F1"):
		return RankCritical
	case strings.HasPrefix(code, "F2"):
		return RankHigh
	case strings.HasPrefix(code, "F3"):
		return RankMedium
	default:
		return RankLow
	}
}

// Escalate reports whether a fault of the given rank needs immediate attention.
func Escalate(r SeverityRank) bool { return r >= RankHigh }

// RankLabel returns a human-readable severity label.
func RankLabel(r SeverityRank) string {
	switch r {
	case RankCritical:
		return "critical"
	case RankHigh:
		return "high"
	case RankMedium:
		return "medium"
	default:
		return "low"
	}
}
""")

add("internal/fault/history.go", """package fault

import (
	"sync"
	"time"
)

// HistoryEntry records one state change for audit and diagnostics.
type HistoryEntry struct {
	FaultID   string
	From      State
	To        State
	At        time.Time
	Note      string
}

// History stores fault transition history in memory.
type History struct {
	mu      sync.RWMutex
	entries []HistoryEntry
}

func NewHistory() *History { return &History{} }

func (h *History) Append(e HistoryEntry) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.entries = append(h.entries, e)
}

func (h *History) ForFault(id string) []HistoryEntry {
	h.mu.RLock()
	defer h.mu.RUnlock()
	var out []HistoryEntry
	for _, e := range h.entries {
		if e.FaultID == id {
			out = append(out, e)
		}
	}
	return out
}

// Timeline returns a copy of all transitions, oldest first.
func (h *History) Timeline() []HistoryEntry {
	h.mu.RLock()
	defer h.mu.RUnlock()
	out := make([]HistoryEntry, len(h.entries))
	copy(out, h.entries)
	return out
}
""")

# --- alarm additions ---
add("internal/alarm/throttle.go", """package alarm

import "time"

// Throttle limits how many alarms a single turbine may raise per window.
type Throttle struct {
	window  time.Duration
	limit   int
	buckets map[string][]time.Time
}

func NewThrottle(window time.Duration, limit int) *Throttle {
	if window <= 0 {
		window = time.Minute
	}
	return &Throttle{window: window, limit: limit, buckets: map[string][]time.Time{}}
}

// Allow records a dispatch attempt and reports whether it is within budget.
func (t *Throttle) Allow(turbineID string, now time.Time) bool {
	times := t.buckets[turbineID]
	cutoff := now.Add(-t.window)
	kept := times[:0]
	for _, ts := range times {
		if ts.After(cutoff) {
			kept = append(kept, ts)
		}
	}
	if len(kept) >= t.limit {
		t.buckets[turbineID] = kept
		return false
	}
	kept = append(kept, now)
	t.buckets[turbineID] = kept
	return true
}
""")

add("internal/alarm/routing.go", """package alarm

// Route classifies an alarm severity into a routing tier.
type Route int

const (
	RouteSilent Route = iota
	RouteOps
	RouteEngineer
	RouteCritical
)

func RouteForSeverity(severity string) Route {
	switch severity {
	case "critical":
		return RouteCritical
	case "warning":
		return RouteEngineer
	case "info":
		return RouteOps
	default:
		return RouteSilent
	}
}

// Router selects notifier channels based on severity.
type Router struct {
	channels map[Route][]chan Alarm
}

func NewRouter() *Router { return &Router{channels: map[Route][]chan Alarm{}} }

func (r *Router) Register(route Route, ch chan Alarm) {
	r.channels[route] = append(r.channels[route], ch)
}

func (r *Router) Fanout(a Alarm) int {
	route := RouteForSeverity(a.Severity)
	sent := 0
	for _, ch := range r.channels[route] {
		select {
		case ch <- a:
			sent++
		default:
		}
	}
	return sent
}
""")

# --- workorder additions ---
add("internal/workorder/parts.go", """package workorder

import "fmt"

// Part is a consumable spare part.
type Part struct {
	SKU      string
	Name     string
	OnHand   int
	Reserved int
}

// Inventory tracks spare parts and reservations.
type Inventory struct {
	parts map[string]*Part
}

func NewInventory(parts []Part) *Inventory {
	inv := &Inventory{parts: map[string]*Part{}}
	for _, p := range parts {
		cp := p
		inv.parts[p.SKU] = &cp
	}
	return inv
}

func (i *Inventory) Available(sku string) int {
	if p, ok := i.parts[sku]; ok {
		return p.OnHand - p.Reserved
	}
	return 0
}

// Reserve decrements available inventory for an order.
func (i *Inventory) Reserve(sku string, qty int) error {
	p, ok := i.parts[sku]
	if !ok {
		return fmt.Errorf("part %s unknown", sku)
	}
	if p.OnHand-p.Reserved < qty {
		return fmt.Errorf("part %s insufficient: have %d need %d", sku, p.OnHand-p.Reserved, qty)
	}
	p.Reserved += qty
	return nil
}

// Release returns reserved parts to availability.
func (i *Inventory) Release(sku string, qty int) {
	if p, ok := i.parts[sku]; ok {
		p.Reserved -= qty
		if p.Reserved < 0 {
			p.Reserved = 0
		}
	}
}
""")

add("internal/workorder/cost.go", """package workorder

// CostEstimate approximates the labor cost of a work order.
type CostEstimate struct {
	OrderID   string
	LaborHours float64
	RatePerHour float64
	PartsCost  float64
	Total      float64
}

// Estimator builds cost estimates from order priority and title length.
type Estimator struct {
	RatePerHour float64
}

func NewEstimator(ratePerHour float64) *Estimator {
	if ratePerHour <= 0 {
		ratePerHour = 120.0
	}
	return &Estimator{RatePerHour: ratePerHour}
}

// Estimate derives a rough labor estimate from order priority.
func (e *Estimator) Estimate(wo WorkOrder, partsCost float64) CostEstimate {
	hours := map[int]float64{1: 2, 2: 4, 3: 8}[wo.Priority]
	if hours == 0 {
		hours = 1
	}
	est := CostEstimate{
		OrderID:     wo.ID,
		LaborHours:  hours,
		RatePerHour: e.RatePerHour,
		PartsCost:   partsCost,
	}
	est.Total = hours*e.RatePerHour + partsCost
	return est
}
""")

# --- analytics additions ---
add("internal/analytics/availability.go", """package analytics

import "windfarm-turbine-monitor-service/internal/telemetry"

// AvailabilityWindow computes turbine availability over a series of samples.
func AvailabilityWindow(samples []telemetry.Sample, minPower float64) float64 {
	if len(samples) == 0 {
		return 0
	}
	up := 0
	for _, s := range samples {
		if s.PowerOutput >= minPower {
			up++
		}
	}
	return float64(up) / float64(len(samples))
}

// CapacityFactor compares mean output to a rated power.
func CapacityFactor(samples []telemetry.Sample, ratedPower float64) float64 {
	if ratedPower <= 0 {
		return 0
	}
	agg := AggregateSamples(samples, 3600)
	return agg.MeanPower / ratedPower
}
""")

add("internal/analytics/correlation.go", """package analytics

import (
	"math"

	"windfarm-turbine-monitor-service/internal/telemetry"
)

// PearsonCorrelation computes the Pearson correlation between wind speed and
// power output over paired samples.
func PearsonCorrelation(samples []telemetry.Sample) float64 {
	n := float64(len(samples))
	if n < 2 {
		return 0
	}
	var sx, sy, sxx, syy, sxy float64
	for _, s := range samples {
		sx += s.WindSpeed
		sy += s.PowerOutput
		sxx += s.WindSpeed * s.WindSpeed
		syy += s.PowerOutput * s.PowerOutput
		sxy += s.WindSpeed * s.PowerOutput
	}
	num := n*sxy - sx*sy
	den := math.Sqrt((n*sxx - sx*sx) * (n*syy - sy*sy))
	if den == 0 {
		return 0
	}
	return num / den
}

// Underperformance flags samples where power is far below the wind-speed curve.
func Underperformance(samples []telemetry.Sample, threshold float64) int {
	count := 0
	for _, s := range samples {
		expected := 0.0
		if s.WindSpeed >= 12 {
			expected = 2000
		} else if s.WindSpeed >= 6 {
			expected = 2000 * (s.WindSpeed - 6) / 6
		}
		if expected > 0 && s.PowerOutput < expected*threshold {
			count++
		}
	}
	return count
}
""")

for rel, c in files.items():
    p = os.path.join(BASE, rel)
    os.makedirs(os.path.dirname(p), exist_ok=True)
    with open(p, "w") as f:
        f.write(c)
print(f"wrote {len(files)} files")

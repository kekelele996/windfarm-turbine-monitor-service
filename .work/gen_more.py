#!/usr/bin/env python3
import os
BASE = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
files = {}
def add(p, c): files[p] = c

# --- telemetry additions ---
add("internal/telemetry/sequence.go", """package telemetry

import (
	"sort"
	"time"
)

// SequenceChecker validates that samples arrive in a sane temporal order and
// detects gaps that indicate a SCADA link outage.
type SequenceChecker struct {
	maxBackfill time.Duration
}

func NewSequenceChecker(maxBackfill time.Duration) *SequenceChecker {
	if maxBackfill <= 0 {
		maxBackfill = 2 * time.Minute
	}
	return &SequenceChecker{maxBackfill: maxBackfill}
}

// Gap is a detected interval without samples for a turbine.
type Gap struct {
	TurbineID  string
	From       time.Time
	To         time.Time
	Duration   time.Duration
}

// SortSamples orders samples by timestamp, oldest first, in place.
func SortSamples(samples []Sample) {
	sort.SliceStable(samples, func(i, j int) bool {
		return samples[i].Timestamp.Before(samples[j].Timestamp)
	})
}

// DetectGaps finds intervals longer than the allowed backfill window in a
// sorted series for a single turbine.
func (c *SequenceChecker) DetectGaps(turbineID string, sorted []Sample) []Gap {
	var gaps []Gap
	if len(sorted) < 2 {
		return gaps
	}
	prev := sorted[0]
	for _, s := range sorted[1:] {
		if s.Timestamp.Sub(prev.Timestamp) > c.maxBackfill {
			gaps = append(gaps, Gap{
				TurbineID: turbineID,
				From:      prev.Timestamp,
				To:        s.Timestamp,
				Duration:  s.Timestamp.Sub(prev.Timestamp),
			})
		}
		prev = s
	}
	return gaps
}

// OutOfOrder reports whether any sample precedes its predecessor.
func OutOfOrder(sorted []Sample) bool {
	for i := 1; i < len(sorted); i++ {
		if sorted[i].Timestamp.Before(sorted[i-1].Timestamp) {
			return true
		}
	}
	return false
}
""")

add("internal/telemetry/batchwriter.go", """package telemetry

import (
	"sync"
	"time"
)

// BatchWriter buffers samples and flushes them to a sink in batches, keeping
// the hot path cheap.
type BatchWriter struct {
	mu      sync.Mutex
	pending []Sample
	limit   int
	flushFn func([]Sample) error
}

func NewBatchWriter(limit int, flushFn func([]Sample) error) *BatchWriter {
	if limit <= 0 {
		limit = 100
	}
	return &BatchWriter{limit: limit, flushFn: flushFn}
}

// Add buffers one sample and flushes when the batch limit is reached.
func (w *BatchWriter) Add(s Sample) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.pending = append(w.pending, s)
	if len(w.pending) >= w.limit {
		return w.flush()
	}
	return nil
}

// Flush writes every pending sample to the sink.
func (w *BatchWriter) Flush() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.flush()
}

func (w *BatchWriter) flush() error {
	if len(w.pending) == 0 {
		return nil
	}
	batch := w.pending
	w.pending = nil
	return w.flushFn(batch)
}

// Pending returns how many samples are currently buffered.
func (w *BatchWriter) Pending() int {
	w.mu.Lock()
	defer w.mu.Unlock()
	return len(w.pending)
}

// PeriodicallyFlush calls Flush at the given interval until stop is closed.
func (w *BatchWriter) PeriodicallyFlush(interval time.Duration, stop <-chan struct{}) {
	t := time.NewTicker(interval)
	defer t.Stop()
	for {
		select {
		case <-stop:
			_ = w.Flush()
			return
		case <-t.C:
			_ = w.Flush()
		}
	}
}
""")

# --- turbine additions ---
add("internal/turbine/performance.go", """package turbine

import "windfarm-turbine-monitor-service/internal/telemetry"

// Derating describes a temporary power cap applied to a turbine.
type Derating struct {
	TurbineID string
	Reason    string
	LimitKW   float64
}

// DeratingTable maps turbines to their current derating limits.
type DeratingTable struct {
	limits map[string]Derating
}

func NewDeratingTable() *DeratingTable { return &DeratingTable{limits: map[string]Derating{}} }

func (t *DeratingTable) Set(d Derating) { t.limits[d.TurbineID] = d }

func (t *DeratingTable) Get(turbineID string) (Derating, bool) {
	d, ok := t.limits[turbineID]
	return d, ok
}

// EffectivePower clamps a measured output to the derating limit.
func (t *DeratingTable) EffectivePower(turbineID string, measured float64) float64 {
	if d, ok := t.limits[turbineID]; ok && measured > d.LimitKW {
		return d.LimitKW
	}
	return measured
}

// Availability computes the fraction of samples where the turbine produced at
// least minPower, which approximates operational availability.
func Availability(samples []telemetry.Sample, minPower float64) float64 {
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
""")

add("internal/turbine/geo.go", """package turbine

import "math"

// Position is a WGS-84 coordinate for a turbine.
type Position struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

// DistanceMeters computes the great-circle distance between two positions.
func DistanceMeters(a, b Position) float64 {
	const earthRadius = 6371000.0
	lat1 := a.Latitude * math.Pi / 180
	lat2 := b.Latitude * math.Pi / 180
	dLat := (b.Latitude - a.Latitude) * math.Pi / 180
	dLon := (b.Longitude - a.Longitude) * math.Pi / 180
	sinLat := math.Sin(dLat / 2)
	sinLon := math.Sin(dLon / 2)
	h := sinLat*sinLat + math.Cos(lat1)*math.Cos(lat2)*sinLon*sinLon
	return 2 * earthRadius * math.Asin(math.Sqrt(h))
}

// ClosestSite finds the nearest named site to a position.
type SiteLocation struct {
	Name string
	Pos  Position
}

func ClosestSite(pos Position, sites []SiteLocation) string {
	if len(sites) == 0 {
		return ""
	}
	best := sites[0]
	bestDist := DistanceMeters(pos, best.Pos)
	for _, s := range sites[1:] {
		d := DistanceMeters(pos, s.Pos)
		if d < bestDist {
			best = s
			bestDist = d
		}
	}
	return best.Name
}
""")

# --- ruleengine additions ---
add("internal/ruleengine/hysteresis.go", """package ruleengine

// Hysteresis prevents alarm flapping by requiring a threshold to be crossed
// with a margin before the alarm clears.
type Hysteresis struct {
	OnThreshold  float64
	OffThreshold float64
	active       map[string]bool
}

func NewHysteresis(on, off float64) *Hysteresis {
	return &Hysteresis{OnThreshold: on, OffThreshold: off, active: map[string]bool{}}
}

// Evaluate updates the latched state for a subject and reports whether the
// condition is currently active.
func (h *Hysteresis) Evaluate(subject string, value float64) bool {
	if h.active[subject] {
		if value <= h.OffThreshold {
			h.active[subject] = false
			return false
		}
		return true
	}
	if value >= h.OnThreshold {
		h.active[subject] = true
		return true
	}
	return false
}

// LatchedSubjects returns every subject currently held active.
func (h *Hysteresis) LatchedSubjects() []string {
	out := make([]string, 0, len(h.active))
	for s, active := range h.active {
		if active {
			out = append(out, s)
		}
	}
	return out
}
""")

add("internal/ruleengine/composite.go", """package ruleengine

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

func NewCompositeRegistry() *CompositeRegistry { return &CompositeRegistry{rules: map[string]CompositeRule{}} }

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
""")

for rel, c in files.items():
    p = os.path.join(BASE, rel)
    os.makedirs(os.path.dirname(p), exist_ok=True)
    with open(p, "w") as f:
        f.write(c)
print(f"wrote {len(files)} files")

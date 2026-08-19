#!/usr/bin/env python3
import os
BASE = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
files = {}
def add(p, c): files[p] = c

add("internal/telemetry/windrose.go", """package telemetry

import "math"

// WindDirectionBucket buckets wind direction into 16 compass sectors.
type WindDirectionBucket struct {
	Sector string
	Count  int
	MeanWind float64
}

// WindRose aggregates samples into direction sectors.
func WindRose(samples []Sample) []WindDirectionBucket {
	buckets := map[string]*WindDirectionBucket{}
	labels := [16]string{"N", "NNE", "NE", "ENE", "E", "ESE", "SE", "SSE", "S", "SSW", "SW", "WSW", "W", "WNW", "NW", "NNW"}
	for _, s := range samples {
		idx := int((s.YawAngle+22.5)/45.0) % 16
		if idx < 0 {
			idx += 16
		}
		label := labels[idx]
		b := buckets[label]
		if b == nil {
			b = &WindDirectionBucket{Sector: label}
			buckets[label] = b
		}
		b.Count++
		b.MeanWind += s.WindSpeed
	}
	out := make([]WindDirectionBucket, 0, len(buckets))
	for _, b := range buckets {
		if b.Count > 0 {
			b.MeanWind /= float64(b.Count)
		}
		out = append(out, *b)
	}
	return out
}

// PrevailingDirection returns the sector with the most samples.
func PrevailingDirection(samples []Sample) string {
	rose := WindRose(samples)
	best := ""
	bestN := -1
	for _, b := range rose {
		if b.Count > bestN {
			best = b.Sector
			bestN = b.Count
		}
	}
	return best
}

// TurbulenceIntensity approximates turbulence from wind speed variance.
func TurbulenceIntensity(samples []Sample) float64 {
	if len(samples) < 2 {
		return 0
	}
	mean := 0.0
	for _, s := range samples {
		mean += s.WindSpeed
	}
	mean /= float64(len(samples))
	if mean == 0 {
		return 0
	}
	var variance float64
	for _, s := range samples {
		d := s.WindSpeed - mean
		variance += d * d
	}
	return math.Sqrt(variance/float64(len(samples))) / mean
}
""")

add("internal/analytics/ramp.go", """package analytics

import "windfarm-turbine-monitor-service/internal/telemetry"

// RampEvent is a rapid change in power output between consecutive samples.
type RampEvent struct {
	TurbineID string
	From      float64
	To        float64
	Delta     float64
}

// DetectRamps finds power changes exceeding a threshold between consecutive
// samples, which can indicate grid events or control failures.
func DetectRamps(samples []telemetry.Sample, thresholdKW float64) []RampEvent {
	var events []RampEvent
	for i := 1; i < len(samples); i++ {
		delta := samples[i].PowerOutput - samples[i-1].PowerOutput
		if delta < 0 {
			delta = -delta
		}
		if delta >= thresholdKW {
			events = append(events, RampEvent{
				TurbineID: samples[i].TurbineID,
				From:      samples[i-1].PowerOutput,
				To:        samples[i].PowerOutput,
				Delta:     delta,
			})
		}
	}
	return events
}

// MaxRamp returns the largest power change in the series.
func MaxRamp(samples []telemetry.Sample) float64 {
	max := 0.0
	for i := 1; i < len(samples); i++ {
		d := samples[i].PowerOutput - samples[i-1].PowerOutput
		if d < 0 {
			d = -d
		}
		if d > max {
			max = d
		}
	}
	return max
}
""")

add("internal/ops/backpressure.go", """package ops

import "sync"

// BackpressureGate limits concurrent processing to a fixed number of slots.
type BackpressureGate struct {
	slots chan struct{}
}

func NewBackpressureGate(limit int) *BackpressureGate {
	if limit <= 0 {
		limit = 8
	}
	return &BackpressureGate{slots: make(chan struct{}, limit)}
}

// Acquire blocks until a slot is available.
func (g *BackpressureGate) Acquire() { g.slots <- struct{}{} }

// Release returns one slot.
func (g *BackpressureGate) Release() { <-g.slots }

// TryAcquire acquires a slot without blocking.
func (g *BackpressureGate) TryAcquire() bool {
	select {
	case g.slots <- struct{}{}:
		return true
	default:
		return false
	}
}

// Available returns how many slots are currently free.
func (g *BackpressureGate) Available() int { return cap(g.slots) - len(g.slots) }

// Run executes fn under a slot and releases it afterwards.
func (g *BackpressureGate) Run(fn func()) {
	g.Acquire()
	defer g.Release()
	fn()
}

// WaitGroupGate is a simpler cooperative gate using a WaitGroup.
type WaitGroupGate struct {
	mu sync.Mutex
	wg sync.WaitGroup
	n  int
}

func (g *WaitGroupGate) Inc() {
	g.mu.Lock()
	g.n++
	g.wg.Add(1)
	g.mu.Unlock()
}

func (g *WaitGroupGate) Done() {
	g.mu.Lock()
	g.n--
	g.wg.Done()
	g.mu.Unlock()
}

func (g *WaitGroupGate) Wait() { g.wg.Wait() }

func (g *WaitGroupGate) Count() int {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.n
}
""")

add("internal/turbine/decommission.go", """package turbine

import "time"

// DecommissionWorkflow tracks the stages of retiring a turbine.
type DecommissionWorkflow struct {
	stages map[string][]Stage
}

type Stage struct {
	TurbineID string
	Name      string
	Done      bool
	At        time.Time
}

func NewDecommissionWorkflow() *DecommissionWorkflow {
	return &DecommissionWorkflow{stages: map[string][]Stage{}}
}

// Start creates the standard decommission stage sequence for a turbine.
func (w *DecommissionWorkflow) Start(turbineID string, now time.Time) {
	names := []string{"grid-disconnect", "blade-removal", "tower-demolition", "site-restore"}
	stages := make([]Stage, 0, len(names))
	for _, n := range names {
		stages = append(stages, Stage{TurbineID: turbineID, Name: n})
	}
	w.stages[turbineID] = stages
}

// CompleteStage marks a stage done.
func (w *DecommissionWorkflow) CompleteStage(turbineID, name string, now time.Time) bool {
	stages := w.stages[turbineID]
	for i := range stages {
		if stages[i].Name == name && !stages[i].Done {
			stages[i].Done = true
			stages[i].At = now
			return true
		}
	}
	return false
}

// Progress returns the fraction of completed stages.
func (w *DecommissionWorkflow) Progress(turbineID string) float64 {
	stages := w.stages[turbineID]
	if len(stages) == 0 {
		return 0
	}
	done := 0
	for _, s := range stages {
		if s.Done {
			done++
		}
	}
	return float64(done) / float64(len(stages))
}
""")

for rel, c in files.items():
    p = os.path.join(BASE, rel)
    os.makedirs(os.path.dirname(p), exist_ok=True)
    with open(p, "w") as f:
        f.write(c)
print(f"wrote {len(files)} files")

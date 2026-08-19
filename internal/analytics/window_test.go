package analytics

import (
	"testing"

	"windfarm-turbine-monitor-service/internal/telemetry"
)

func TestWindowSnapshotStaysPut(t *testing.T) {
	w := NewSlidingWindow(2)
	w.Add(1)
	w.Add(2)
	snap := w.Values()
	w.Add(3)
	if len(snap) != 2 || snap[0] != 1 || snap[1] != 2 {
		t.Fatalf("snapshot mutated: %v", snap)
	}
}

func TestWindowCountAccurate(t *testing.T) {
	w := NewSlidingWindow(8)
	w.Add(1)
	w.Add(2)
	w.Add(3)
	if w.Count() != 3 {
		t.Fatalf("count %d want 3", w.Count())
	}
}

func TestWindowMeanAccurate(t *testing.T) {
	w := NewSlidingWindow(8)
	w.Add(2)
	w.Add(4)
	if got := w.Mean(); got != 3 {
		t.Fatalf("mean %v want 3", got)
	}
}

func TestFilterSamplesNoMutate(t *testing.T) {
	samples := []telemetry.Sample{{PowerOutput: 1}, {PowerOutput: 9}, {PowerOutput: 5}}
	_ = FilterSamples(samples, 4)
	if len(samples) != 3 || samples[0].PowerOutput != 1 || samples[1].PowerOutput != 9 || samples[2].PowerOutput != 5 {
		t.Fatalf("input mutated: %+v", samples)
	}
}

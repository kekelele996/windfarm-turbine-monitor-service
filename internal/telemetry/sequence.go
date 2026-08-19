package telemetry

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
	TurbineID string
	From      time.Time
	To        time.Time
	Duration  time.Duration
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

package telemetry

import (
	"sync"
	"time"

	"windfarm-turbine-monitor-service/internal/platform"
)

// SampleRegistry keeps a bounded, per-turbine ring of recent samples.
type SampleRegistry struct {
	mu       sync.RWMutex
	capacity int
	bufs     map[string][]Sample
}

func NewSampleRegistry(capacity int) *SampleRegistry {
	if capacity <= 0 {
		capacity = 512
	}
	return &SampleRegistry{capacity: capacity, bufs: map[string][]Sample{}}
}

func (r *SampleRegistry) Append(s Sample) error {
	if s.TurbineID == "" {
		return platform.ErrInvalid
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	buf := r.bufs[s.TurbineID]
	buf = append(buf, s)
	if len(buf) > r.capacity {
		buf = buf[len(buf)-r.capacity:]
	}
	r.bufs[s.TurbineID] = buf
	return nil
}

// Recent returns the most recent n samples for a turbine.
func (r *SampleRegistry) Recent(turbineID string, n int) []Sample {
	r.mu.RLock()
	defer r.mu.RUnlock()
	buf := r.bufs[turbineID]
	if len(buf) < n {
		n = len(buf)
	}
	return buf[len(buf)-n:]
}

// Latest returns the most recent sample for a turbine and whether one exists.
func (r *SampleRegistry) Latest(turbineID string) (Sample, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	buf := r.bufs[turbineID]
	if len(buf) == 0 {
		return Sample{}, false
	}
	return buf[len(buf)-1], true
}

// Since returns samples with a timestamp at or after cutoff, oldest first.
func (r *SampleRegistry) Since(turbineID string, cutoff time.Time) []Sample {
	r.mu.RLock()
	defer r.mu.RUnlock()
	buf := r.bufs[turbineID]
	out := buf[:0]
	for _, s := range buf {
		if !s.Timestamp.Before(cutoff) {
			out = append(out, s)
		}
	}
	return out
}

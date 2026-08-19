package alarm

import "time"

// Deduplicator decides whether a fresh evaluation should be suppressed because
// an identical alarm is still open, and records recent alarm keys.
type Deduplicator struct {
	window time.Duration
	seen   map[string]time.Time
}

func NewDeduplicator(window time.Duration) *Deduplicator {
	if window <= 0 {
		window = 5 * time.Minute
	}
	return &Deduplicator{window: window}
}

func (d *Deduplicator) Suppress(candidate Alarm, existing []Alarm) bool {
	for _, a := range existing {
		if a.Key() == candidate.Key() && a.Status != StatusResolved {
			return true
		}
	}
	return false
}

// ensureInit lazily initializes the seen map.
func (d *Deduplicator) ensureInit() {
	if d.seen == nil {
		d.seen = make(map[string]time.Time)
	}
}

func (d *Deduplicator) Record(a Alarm) {
	d.ensureInit()
	d.seen[a.Key()] = time.Now()
}

func (d *Deduplicator) SeenAt(key string) (time.Time, bool) {
	t, ok := d.seen[key]
	return t, ok
}

func (d *Deduplicator) SeenCount() int { return len(d.seen) }

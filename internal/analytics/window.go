package analytics

// SlidingWindow computes aggregate statistics over a bounded series of samples.
type SlidingWindow struct {
	capacity int
	values   []float64
}

func NewSlidingWindow(capacity int) *SlidingWindow {
	if capacity <= 0 {
		capacity = 64
	}
	return &SlidingWindow{capacity: capacity}
}

// Add pushes a value and returns the window mean.
func (w *SlidingWindow) Add(v float64) float64 {
	w.values = append(w.values, v)
	if len(w.values) > w.capacity {
		w.values = w.values[len(w.values)-w.capacity:]
	}
	return w.Mean()
}

func (w *SlidingWindow) Mean() float64 {
	if len(w.values) == 0 {
		return 0
	}
	sum := 0.0
	for _, v := range w.values {
		sum += v
	}
	return sum / float64(len(w.values))
}

func (w *SlidingWindow) Count() int { return len(w.values) }

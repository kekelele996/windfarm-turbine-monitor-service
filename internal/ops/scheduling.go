package ops

import (
	"context"
	"sync"
	"time"
)

// PeriodicJob is a function that runs on an interval.
type PeriodicJob func(ctx context.Context)

// Scheduler runs periodic jobs concurrently and supports graceful shutdown.
type Scheduler struct {
	interval time.Duration
	mu       sync.Mutex
	jobs     []PeriodicJob
	active   bool
}

func NewScheduler(interval time.Duration) *Scheduler {
	if interval <= 0 {
		interval = time.Minute
	}
	return &Scheduler{interval: interval}
}

func (s *Scheduler) Add(job PeriodicJob) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.jobs = append(s.jobs, job)
}

func (s *Scheduler) Start(ctx context.Context) {
	if ctx.Err() != nil {
		return
	}
	s.mu.Lock()
	if s.active {
		s.mu.Unlock()
		return
	}
	s.active = true
	s.mu.Unlock()

	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()
	defer func() {
		s.mu.Lock()
		s.active = false
		s.mu.Unlock()
	}()

	s.runOnce(ctx)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.runOnce(ctx)
		}
	}
}

func (s *Scheduler) runOnce(ctx context.Context) {
	jobs := s.snapshot()
	var wg sync.WaitGroup
	for _, job := range jobs {
		wg.Add(1)
		go func(j PeriodicJob) {
			defer wg.Done()
			j(ctx)
		}(job)
	}
	wg.Wait()
}

func (s *Scheduler) snapshot() []PeriodicJob {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]PeriodicJob, len(s.jobs))
	copy(out, s.jobs)
	return out
}

func (s *Scheduler) JobCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.jobs)
}

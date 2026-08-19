package telemetry

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

type countingStore struct {
	mu     sync.Mutex
	n      int
	failAt int
}

func (c *countingStore) Append(s Sample) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.n++
	if c.failAt > 0 && c.n >= c.failAt {
		return errors.New("sink full")
	}
	return nil
}

func (c *countingStore) count() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.n
}

func TestBatchIngestKeepsCount(t *testing.T) {
	store := &countingStore{}
	in := NewIngestor(store, 4)
	samples := make([]Sample, 300)
	for i := range samples {
		samples[i] = Sample{TurbineID: "WTG-001"}
	}
	start := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(2)
	go func() { defer wg.Done(); <-start; _, _ = in.Ingest(context.Background(), samples[:150]) }()
	go func() { defer wg.Done(); <-start; _, _ = in.Ingest(context.Background(), samples[150:]) }()
	close(start)
	wg.Wait()
	if got := store.count(); got != len(samples) {
		t.Fatalf("store appended %d want %d", got, len(samples))
	}
}

func TestBatchIngestReportsSinkError(t *testing.T) {
	store := &countingStore{failAt: 1}
	in := NewIngestor(store, 4)
	samples := make([]Sample, 50)
	for i := range samples {
		samples[i] = Sample{TurbineID: "WTG-001"}
	}
	start := make(chan struct{})
	var wg sync.WaitGroup
	var mu sync.Mutex
	var gotErr error
	wg.Add(2)
	go func() {
		defer wg.Done()
		<-start
		if _, err := in.Ingest(context.Background(), samples); err != nil {
			mu.Lock()
			gotErr = err
			mu.Unlock()
		}
	}()
	go func() {
		defer wg.Done()
		<-start
		if _, err := in.Ingest(context.Background(), samples); err != nil {
			mu.Lock()
			gotErr = err
			mu.Unlock()
		}
	}()
	close(start)
	wg.Wait()
	if gotErr == nil {
		t.Fatalf("expected sink error, got nil")
	}
}


func TestBatchWriterFlushesOnStop(t *testing.T) {
	store := &countingStore{}
	bw := NewBatchWriter(1000, func(batch []Sample) error {
		for _, s := range batch {
			_ = store.Append(s)
		}
		return nil
	})
	stop := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		<-stop
	}()
	go func() {
		defer wg.Done()
		_ = bw.Add(Sample{TurbineID: "WTG-001"})
		time.Sleep(10 * time.Millisecond)
		bw.PeriodicallyFlush(time.Hour, stop)
	}()
	close(stop)
	wg.Wait()
	if store.count() == 0 {
		t.Fatalf("samples not flushed on stop")
	}
}

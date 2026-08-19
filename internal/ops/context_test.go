package ops

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"windfarm-turbine-monitor-service/internal/alarm"
	"windfarm-turbine-monitor-service/internal/audit"
	"windfarm-turbine-monitor-service/internal/fault"
	"windfarm-turbine-monitor-service/internal/ruleengine"
	"windfarm-turbine-monitor-service/internal/telemetry"
)

func newTestPipeline() *Pipeline {
	samples := telemetry.NewSampleRegistry(128)
	ingestor := telemetry.NewIngestor(samples, 4)
	rules := ruleengine.NewRegistry(nil)
	evaluator := ruleengine.NewEvaluator(rules)
	dedup := alarm.NewDeduplicator(time.Minute)
	dispatcher := alarm.NewDispatcher(dedup)
	notifier := alarm.NewNotifier()
	faults := fault.NewService(fault.NewStore())
	stream := audit.NewStream()
	return &Pipeline{Ingestor: ingestor, Evaluator: evaluator, Alarms: dispatcher, Notifier: notifier, Faults: faults, Audit: stream}
}

func TestIngestStopsOnCancel(t *testing.T) {
	p := newTestPipeline()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := p.ProcessIngest(ctx, telemetry.SampleBatch{
		Source:  "test",
		Samples: []telemetry.Sample{{TurbineID: "WTG-001"}},
	})
	if err == nil {
		t.Fatalf("expected cancellation error, got nil")
	}
}

func TestSchedulerSkipsOnCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	s := NewScheduler(time.Hour)
	var ran int32
	s.Add(func(ctx context.Context) { atomic.AddInt32(&ran, 1) })
	s.Start(ctx)
	if got := atomic.LoadInt32(&ran); got != 0 {
		t.Fatalf("jobs ran after cancellation: %d", got)
	}
}

func TestSchedulerJobGetsCtx(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	s := NewScheduler(time.Hour)
	gotCancel := false
	s.Add(func(c context.Context) {
		if c.Err() != nil {
			gotCancel = true
		}
	})
	s.runOnce(ctx)
	if !gotCancel {
		t.Fatalf("job did not receive cancelled context")
	}
}

func TestRunContextHonorsCancel(t *testing.T) {
	g := NewBackpressureGate(4)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	var ran int32
	g.RunContext(ctx, func() { atomic.AddInt32(&ran, 1) })
	if got := atomic.LoadInt32(&ran); got != 0 {
		t.Fatalf("fn ran after cancellation: %d", got)
	}
}

#!/usr/bin/env python3
import os
BASE = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
files = {}
def add(p, c): files[p] = c

add("internal/ops/pipeline.go", """package ops

import (
	"context"
	"fmt"

	"windfarm-turbine-monitor-service/internal/alarm"
	"windfarm-turbine-monitor-service/internal/audit"
	"windfarm-turbine-monitor-service/internal/fault"
	"windfarm-turbine-monitor-service/internal/ruleengine"
	"windfarm-turbine-monitor-service/internal/telemetry"
)

// Pipeline orchestrates ingestion through evaluation to alarm/fault creation.
type Pipeline struct {
	Ingestor  *telemetry.Ingestor
	Evaluator *ruleengine.Evaluator
	Alarms    *alarm.Dispatcher
	Notifier  *alarm.Notifier
	Faults    *fault.Service
	Audit     *audit.Stream
}

// ProcessStats reports the outcome of one ingestion pass.
type ProcessStats struct {
	Ingested     int
	Rejected     int
	AlarmsRaised int
	FaultsRaised int
}

func (p *Pipeline) ProcessIngest(ctx context.Context, batch telemetry.SampleBatch) (ProcessStats, error) {
	var stats ProcessStats
	validator := telemetry.DefaultValidator()
	valid, errs := validator.ValidateBatch(batch)
	stats.Rejected = len(errs)

	normalizer := telemetry.Normalizer{}
	normalizer.NormalizeBatch(valid)

	ingested, err := p.Ingestor.Ingest(ctx, valid)
	if err != nil {
		return stats, fmt.Errorf("ingest: %w", err)
	}
	stats.Ingested = ingested

	for _, s := range valid {
		for _, ev := range p.Evaluator.Evaluate(s) {
			candidate := alarm.Alarm{
				TurbineID: s.TurbineID,
				RuleID:    ev.Rule.ID,
				Metric:    string(ev.Rule.Metric),
				Severity:  string(ev.Rule.Severity),
				Value:     ev.Value,
				Threshold: ev.Rule.Threshold,
			}
			created, ok, _ := p.Alarms.Dispatch(candidate)
			if !ok {
				continue
			}
			stats.AlarmsRaised++
			p.Notifier.Notify(created)
			p.Audit.Publish(audit.EventAlarm, s.TurbineID, created.RuleID)

			if ev.Rule.Severity == ruleengine.SeverityCritical {
				if _, err := p.Faults.Raise(s.TurbineID, "F1-"+string(ev.Rule.Metric), "critical"); err == nil {
					stats.FaultsRaised++
					p.Audit.Publish(audit.EventFault, s.TurbineID, "F1-"+string(ev.Rule.Metric))
				}
			}
		}
	}
	return stats, nil
}
""")

add("internal/ops/faultflow.go", """package ops

import (
	"fmt"
	"time"

	"windfarm-turbine-monitor-service/internal/audit"
	"windfarm-turbine-monitor-service/internal/fault"
	"windfarm-turbine-monitor-service/internal/workorder"
)

// FaultFlow links a diagnosed fault to a maintenance work order and tracks the
// handoff between the fault state machine and scheduling.
type FaultFlow struct {
	Faults *fault.Service
	Orders *workorder.Scheduler
	Audit  *audit.Stream
	now    func() time.Time
}

func NewFaultFlow(faults *fault.Service, orders *workorder.Scheduler, stream *audit.Stream, now func() time.Time) *FaultFlow {
	if now == nil {
		now = time.Now
	}
	return &FaultFlow{Faults: faults, Orders: orders, Audit: stream, now: now}
}

// DispatchWorkOrder converts an open fault into a scheduled work order and
// acknowledges the fault.
func (f *FaultFlow) DispatchWorkOrder(faultID string, priority int) (workorder.WorkOrder, fault.Fault, error) {
	flt, err := f.Faults.Get(faultID)
	if err != nil {
		return workorder.WorkOrder{}, fault.Fault{}, fmt.Errorf("get fault: %w", err)
	}
	if flt.State == fault.StateNormal {
		return workorder.WorkOrder{}, fault.Fault{}, fmt.Errorf("fault already resolved")
	}
	wo := f.Orders.Schedule(flt.TurbineID, "Repair "+flt.Code, priority)
	updated, err := f.Faults.Acknowledge(faultID)
	if err != nil {
		return workorder.WorkOrder{}, fault.Fault{}, fmt.Errorf("acknowledge fault: %w", err)
	}
	f.Audit.Publish(audit.EventWorkOrder, flt.TurbineID, wo.ID)
	return wo, updated, nil
}

// ResolveAfterWork marks a fault resolved once its work order completes.
func (f *FaultFlow) ResolveAfterWork(faultID string) (fault.Fault, error) {
	return f.Faults.Resolve(faultID)
}

// OpenCritical returns all open faults ranked critical.
func (f *FaultFlow) OpenCritical() []fault.Fault {
	var out []fault.Fault
	for _, flt := range f.Faults.Open() {
		if fault.SeverityForCode(flt.Code) == fault.RankCritical {
			out = append(out, flt)
		}
	}
	return out
}
""")

add("internal/ops/dispatch.go", """package ops

import (
	"sync"
	"time"

	"windfarm-turbine-monitor-service/internal/alarm"
	"windfarm-turbine-monitor-service/internal/audit"
)

// AlarmDispatch routes alarms through notifier channels and applies throttling
// and escalation timers.
type AlarmDispatch struct {
	Dispatcher *alarm.Dispatcher
	Router     *alarm.Router
	Throttle   *alarm.Throttle
	Audit      *audit.Stream
	mu         sync.Mutex
}

func NewAlarmDispatch(d *alarm.Dispatcher, r *alarm.Router, t *alarm.Throttle, stream *audit.Stream) *AlarmDispatch {
	return &AlarmDispatch{Dispatcher: d, Router: r, Throttle: t, Audit: stream}
}

// DispatchAndRoute applies throttle, dispatches and routes the alarm.
func (a *AlarmDispatch) DispatchAndRoute(candidate alarm.Alarm, now time.Time) (alarm.Alarm, bool) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if !a.Throttle.Allow(candidate.TurbineID, now) {
		return alarm.Alarm{}, false
	}
	created, ok, _ := a.Dispatcher.Dispatch(candidate)
	if !ok {
		return alarm.Alarm{}, false
	}
	if a.Router != nil {
		a.Router.Fanout(created)
	}
	if a.Audit != nil {
		a.Audit.Publish(audit.EventAlarm, created.TurbineID, created.RuleID)
	}
	return created, true
}

// Escalate moves long-open alarms to a higher routing tier.
func (a *AlarmDispatch) Escalate(age time.Duration) int {
	count := 0
	for _, al := range a.Dispatcher.ListOpen() {
		if time.Since(al.OccurredAt) > age {
			_ = a.Dispatcher.Ack(al.ID)
			count++
		}
	}
	return count
}
""")

add("internal/ops/reporter.go", """package ops

import (
	"time"

	"windfarm-turbine-monitor-service/internal/report"
)

// ReportRunner generates site reports on a schedule.
type ReportRunner struct {
	Builder *report.Builder
	now     func() time.Time
}

func NewReportRunner(b *report.Builder, now func() time.Time) *ReportRunner {
	if now == nil {
		now = time.Now
	}
	return &ReportRunner{Builder: b, now: now}
}

// LatestSiteReports produces a report for every known site over the last day.
func (r *ReportRunner) LatestSiteReports() map[string]report.Report {
	out := map[string]report.Report{}
	now := r.now()
	for _, t := range r.Builder.Turbines().List() {
		if _, done := out[t.Site]; done {
			continue
		}
		out[t.Site] = r.Builder.Build(t.Site, now.Add(-24*time.Hour), now)
	}
	return out
}
""")

for rel, c in files.items():
    p = os.path.join(BASE, rel)
    os.makedirs(os.path.dirname(p), exist_ok=True)
    with open(p, "w") as f:
        f.write(c)
print(f"wrote {len(files)} files")

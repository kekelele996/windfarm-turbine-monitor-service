package ops

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

	ingested, err := p.Ingestor.Ingest(context.Background(), valid)
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

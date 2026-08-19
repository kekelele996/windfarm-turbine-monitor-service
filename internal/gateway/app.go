package gateway

import (
	"windfarm-turbine-monitor-service/internal/alarm"
	"windfarm-turbine-monitor-service/internal/audit"
	"windfarm-turbine-monitor-service/internal/fault"
	"windfarm-turbine-monitor-service/internal/report"
	"windfarm-turbine-monitor-service/internal/ruleengine"
	"windfarm-turbine-monitor-service/internal/telemetry"
	"windfarm-turbine-monitor-service/internal/turbine"
	"windfarm-turbine-monitor-service/internal/workorder"
)

// App bundles all domain services behind the HTTP layer.
type App struct {
	Turbines  *turbine.Registry
	Samples   *telemetry.SampleRegistry
	Ingestor  *telemetry.Ingestor
	Rules     *ruleengine.Registry
	Evaluator *ruleengine.Evaluator
	Faults    *fault.Service
	Alarms    *alarm.Dispatcher
	Notifier  *alarm.Notifier
	Orders    *workorder.Scheduler
	Audit     *audit.Stream
	Reports   *report.Builder
}

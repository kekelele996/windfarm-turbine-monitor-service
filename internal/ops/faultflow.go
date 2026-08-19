package ops

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

package workorder

import (
	"testing"

	"windfarm-turbine-monitor-service/internal/fault"
)

func TestRetryingFaultBackToNormal(t *testing.T) {
	store := fault.NewStore()
	f, err := store.Create(fault.Fault{TurbineID: "WTG-001", Code: "F1-gear", State: fault.StateWarning})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Transition(f.ID, fault.StateFault); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Transition(f.ID, fault.StateRetrying); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Transition(f.ID, fault.StateNormal); err != nil {
		t.Fatalf("retrying -> normal should be allowed: %v", err)
	}
}

func TestOrderLifecycleCompletes(t *testing.T) {
	l := NewLifecycle([]WorkOrder{{ID: "W1", Status: StatusScheduled}}, nil)
	if _, err := l.Start("W1"); err != nil {
		t.Fatalf("start failed: %v", err)
	}
	if _, err := l.Complete("W1"); err != nil {
		t.Fatalf("complete failed: %v", err)
	}
}

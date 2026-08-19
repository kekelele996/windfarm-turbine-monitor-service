package fault

import (
	"errors"
	"testing"

	"windfarm-turbine-monitor-service/internal/platform"
)

func TestMissingFaultKeepsNotFound(t *testing.T) {
	svc := NewService(NewStore())
	_, err := svc.Resolve("missing-fault")
	if !errors.Is(err, platform.ErrNotFound) {
		t.Fatalf("expected ErrNotFound in chain, got: %v", err)
	}
}

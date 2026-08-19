package report

import (
	"testing"
	"time"

	"windfarm-turbine-monitor-service/internal/telemetry"
)

func TestSinceDoesNotCorruptRegistry(t *testing.T) {
	r := telemetry.NewSampleRegistry(8)
	base := time.Now()
	for i := 0; i < 3; i++ {
		_ = r.Append(telemetry.Sample{TurbineID: "T1", Timestamp: base.Add(time.Duration(i) * time.Second)})
	}
	_ = r.Since("T1", base.Add(time.Second))
	rec := r.Recent("T1", 10)
	if len(rec) != 3 || !rec[0].Timestamp.Equal(base) {
		t.Fatalf("registry corrupted: %+v", rec)
	}
}

func TestRecentCallerMutationIsolated(t *testing.T) {
	r := telemetry.NewSampleRegistry(8)
	_ = r.Append(telemetry.Sample{TurbineID: "T1", WindSpeed: 10})
	rec := r.Recent("T1", 1)
	rec[0].WindSpeed = 999
	again := r.Recent("T1", 1)
	if again[0].WindSpeed == 999 {
		t.Fatalf("caller mutation leaked into registry")
	}
}

func TestFilterWithinKeepsInput(t *testing.T) {
	t0 := time.Unix(100, 0)
	t1 := time.Unix(200, 0)
	t2 := time.Unix(300, 0)
	samples := []telemetry.Sample{{Timestamp: t0}, {Timestamp: t1}, {Timestamp: t2}}
	_ = FilterWithin(samples, time.Unix(150, 0), time.Unix(250, 0))
	if len(samples) != 3 || !samples[0].Timestamp.Equal(t0) {
		t.Fatalf("input mutated: %+v", samples)
	}
}

func TestSortRowsKeepsInput(t *testing.T) {
	sec := Section{Rows: []SectionRow{{Key: "b"}, {Key: "a"}}}
	_ = SortRows(sec)
	if sec.Rows[0].Key != "b" || sec.Rows[1].Key != "a" {
		t.Fatalf("input section mutated: %+v", sec.Rows)
	}
}

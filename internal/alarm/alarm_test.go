package alarm

import (
	"testing"
	"time"
)

func TestDispatchFirstAlarmSafe(t *testing.T) {
	d := NewDispatcher(NewDeduplicator(time.Minute))
	if _, _, err := d.Dispatch(Alarm{TurbineID: "WTG-001", RuleID: "gear-1"}); err != nil {
		t.Fatal(err)
	}
}

func TestRecordFirstKeySafe(t *testing.T) {
	dedup := NewDeduplicator(time.Minute)
	dedup.Record(Alarm{TurbineID: "WTG-001", RuleID: "gear-1"})
	if _, ok := dedup.SeenAt("WTG-001:gear-1"); !ok {
		t.Fatalf("expected key recorded")
	}
}

func TestRegisterFirstRouteSafe(t *testing.T) {
	r := NewRouter()
	r.Register(RouteOps, make(chan Alarm, 1))
	if r.Fanout(Alarm{Severity: "info"}) != 1 {
		t.Fatalf("expected one fanout")
	}
}

func TestTrackFirstAlarmSafe(t *testing.T) {
	n := NewNotifier()
	n.Track(Alarm{TurbineID: "WTG-001"})
	if n.counts["WTG-001"] != 1 {
		t.Fatalf("expected tracked count 1")
	}
}

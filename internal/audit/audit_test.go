package audit

import "testing"

func TestBatchDeliveryInOrder(t *testing.T) {
	s := NewStream()
	s.Publish(EventIngest, "T1", "a")
	s.Publish(EventIngest, "T1", "b")
	var got []string
	if err := s.DeliverBatch(func(e Event) error {
		got = append(got, e.Payload)
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0] != "a" || got[1] != "b" {
		t.Fatalf("events out of order: %v", got)
	}
}

func TestUnsubscribeClosesChannel(t *testing.T) {
	s := NewStream()
	ch := s.Subscribe("sub")
	s.Unsubscribe("sub")
	select {
	case _, ok := <-ch:
		if ok {
			t.Fatalf("channel not closed")
		}
	default:
		t.Fatalf("channel not closed (no data)")
	}
}

func TestBlockedSubscriberErrorKept(t *testing.T) {
	s := NewStream()
	s.Publish(EventIngest, "T1", "a")
	d := NewDelivery(s, NewCheckpoint())
	ch := make(chan Event)
	if err := d.Deliver("sub", ch); err == nil {
		t.Fatalf("expected blocked subscriber error, got nil")
	}
}

func TestAdvanceRegressionRejected(t *testing.T) {
	c := NewCheckpoint()
	c.Set("sub", 5)
	if err := c.Advance("sub", 3); err == nil {
		t.Fatalf("expected regression error, got nil")
	}
}

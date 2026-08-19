package turbine

import "testing"

func seedRegistry() *Registry {
	return NewRegistry([]Turbine{
		{ID: "WTG-001", Name: "Alpha", Site: "NORTH", State: StateRunning},
		{ID: "WTG-002", Name: "Beta", Site: "NORTH", State: StateRunning},
	})
}

func TestPutSeenInList(t *testing.T) {
	r := seedRegistry()
	_ = r.List()
	_ = r.Put(Turbine{ID: "WTG-003", Name: "Gamma", Site: "NORTH", State: StateRunning})
	for _, tt := range r.List() {
		if tt.ID == "WTG-003" {
			return
		}
	}
	t.Fatalf("new turbine missing from List")
}

func TestUpdateStateSeenInListActive(t *testing.T) {
	r := seedRegistry()
	_ = r.List()
	_ = r.UpdateState("WTG-001", StateMaintenance)
	for _, tt := range r.ListActive() {
		if tt.ID == "WTG-001" {
			t.Fatalf("maintenance turbine still active in ListActive")
		}
	}
}

func TestCountMatches(t *testing.T) {
	r := seedRegistry()
	_ = r.List()
	_ = r.Put(Turbine{ID: "WTG-003", State: StateRunning})
	if got := r.Count(); got != 3 {
		t.Fatalf("count %d want 3", got)
	}
}

func TestStatsTotalMatches(t *testing.T) {
	r := seedRegistry()
	_ = r.List()
	_ = r.Put(Turbine{ID: "WTG-003", State: StateRunning})
	if got := r.Stats().Total; got != 3 {
		t.Fatalf("stats total %d want 3", got)
	}
}

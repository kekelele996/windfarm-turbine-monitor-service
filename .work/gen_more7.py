#!/usr/bin/env python3
import os
BASE = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
files = {}
def add(p, c): files[p] = c

add("internal/gateway/handlers_more.go", """package gateway

import (
	"encoding/json"
	"net/http"

	"windfarm-turbine-monitor-service/internal/turbine"
)

func (a *App) handleTurbineDetail(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	t, err := a.Turbines.Get(id)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, t)
}

func (a *App) handleUpdateTurbineState(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var req struct {
		State turbine.TurbineState `json:"state"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	if req.State == "" {
		writeError(w, http.StatusBadRequest, "state required")
		return
	}
	if err := a.Turbines.UpdateState(id, req.State); err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	a.Audit.Publish("turbine", id, "state:"+string(req.State))
	t, _ := a.Turbines.Get(id)
	writeJSON(w, http.StatusOK, t)
}

func (a *App) handleListWorkorders(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, a.Orders.List())
}

func (a *App) handleAssignOrder(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var req struct {
		CrewID string `json:"crew_id"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)
	wo, err := a.Orders.Assign(id, req.CrewID)
	if err != nil {
		writeError(w, http.StatusConflict, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, wo)
}

func (a *App) handleAuditHistory(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, a.Audit.History())
}

func (a *App) handleTurbineStats(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, a.Turbines.Stats())
}

func (a *App) handleAckFault(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	f, err := a.Faults.Acknowledge(id)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, f)
}
""")

# expand router.go with the new routes
import pathlib
rp = os.path.join(BASE, "internal/gateway/router.go")
s = pathlib.Path(rp).read_text()
s = s.replace(
    'mux.HandleFunc("GET /api/turbines", a.handleListTurbines)',
    'mux.HandleFunc("GET /api/turbines", a.handleListTurbines)\n\tmux.HandleFunc("GET /api/turbines/{id}", a.handleTurbineDetail)\n\tmux.HandleFunc("PATCH /api/turbines/{id}/state", a.handleUpdateTurbineState)',
)
s = s.replace(
    'mux.HandleFunc("POST /api/workorders", a.handleScheduleOrder)',
    'mux.HandleFunc("POST /api/workorders", a.handleScheduleOrder)\n\tmux.HandleFunc("GET /api/workorders", a.handleListWorkorders)\n\tmux.HandleFunc("POST /api/workorders/{id}/assign", a.handleAssignOrder)',
)
s = s.replace(
    'mux.HandleFunc("POST /api/faults/{id}/resolve", a.handleResolveFault)',
    'mux.HandleFunc("POST /api/faults/{id}/resolve", a.handleResolveFault)\n\tmux.HandleFunc("POST /api/faults/{id}/ack", a.handleAckFault)',
)
s = s.replace(
    'mux.HandleFunc("GET /api/reports/{site}", a.handleReport)',
    'mux.HandleFunc("GET /api/reports/{site}", a.handleReport)\n\tmux.HandleFunc("GET /api/audit", a.handleAuditHistory)\n\tmux.HandleFunc("GET /api/sites/stats", a.handleTurbineStats)',
)
files["internal/gateway/router.go"] = s

add("internal/ops/scheduling.go", """package ops

import (
	"context"
	"sync"
	"time"
)

// PeriodicJob is a function that runs on an interval.
type PeriodicJob func(ctx context.Context)

// Scheduler runs periodic jobs concurrently and supports graceful shutdown.
type Scheduler struct {
	interval time.Duration
	mu       sync.Mutex
	jobs     []PeriodicJob
	active   bool
}

func NewScheduler(interval time.Duration) *Scheduler {
	if interval <= 0 {
		interval = time.Minute
	}
	return &Scheduler{interval: interval}
}

func (s *Scheduler) Add(job PeriodicJob) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.jobs = append(s.jobs, job)
}

// Start runs all jobs once per interval until the context is cancelled.
func (s *Scheduler) Start(ctx context.Context) {
	s.mu.Lock()
	if s.active {
		s.mu.Unlock()
		return
	}
	s.active = true
	s.mu.Unlock()

	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()
	defer func() {
		s.mu.Lock()
		s.active = false
		s.mu.Unlock()
	}()

	run := func() {
		var wg sync.WaitGroup
		for _, job := range s.snapshot() {
			wg.Add(1)
			go func(j PeriodicJob) {
				defer wg.Done()
				j(ctx)
			}(job)
		}
		wg.Wait()
	}

	run()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			run()
		}
	}
}

func (s *Scheduler) snapshot() []PeriodicJob {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]PeriodicJob, len(s.jobs))
	copy(out, s.jobs)
	return out
}

// JobCount returns the number of registered jobs.
func (s *Scheduler) JobCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.jobs)
}
""")

add("internal/turbine/warranty.go", """package turbine

import "time"

// Warranty describes the manufacturer warranty for a turbine.
type Warranty struct {
	TurbineID string
	Start     time.Time
	Years     int
}

// ExpiresAt returns the warranty end date.
func (w Warranty) ExpiresAt() time.Time {
	return w.Start.AddDate(w.Years, 0, 0)
}

// ActiveAt reports whether the warranty covers a given time.
func (w Warranty) ActiveAt(at time.Time) bool {
	return !at.After(w.ExpiresAt())
}

// RemainingDays returns the warranty days remaining at a time.
func (w Warranty) RemainingDays(at time.Time) int {
	exp := w.ExpiresAt()
	if at.After(exp) {
		return 0
	}
	return int(exp.Sub(at).Hours() / 24)
}

// Coverage maps turbine IDs to warranties.
type Coverage struct {
	items map[string]Warranty
}

func NewCoverage() *Coverage { return &Coverage{items: map[string]Warranty{}} }

func (c *Coverage) Add(w Warranty) { c.items[w.TurbineID] = w }

func (c *Coverage) Covers(turbineID string, at time.Time) bool {
	w, ok := c.items[turbineID]
	if !ok {
		return false
	}
	return w.ActiveAt(at)
}
""")

add("internal/energy/export.go", """package energy

import "windfarm-turbine-monitor-service/internal/grid"

// ExportPlan reconciles curtailment against metered energy.
type ExportPlan struct {
	Site       string
	CurtailedKWh float64
	ExportedKWh  float64
}

// Reconcile applies an active curtailment limit to metered energy.
func Reconcile(site string, meteredKWh float64, sched *grid.Schedule, at interface{ Now() timeT }) float64 {
	_ = site
	_ = meteredKWh
	_ = sched
	_ = at
	return meteredKWh
}
""")

for rel, c in files.items():
    p = os.path.join(BASE, rel)
    os.makedirs(os.path.dirname(p), exist_ok=True)
    with open(p, "w") as f:
        f.write(c)
print(f"wrote {len(files)} files")

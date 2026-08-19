package gateway

import "net/http"

// NewRouter wires all routes to the app handlers using Go 1.22 path values.
func NewRouter(a *App) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("content-type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})
	mux.HandleFunc("GET /api/summary", a.handleSummary)
	mux.HandleFunc("GET /api/turbines", a.handleListTurbines)
	mux.HandleFunc("GET /api/turbines/{id}", a.handleTurbineDetail)
	mux.HandleFunc("PATCH /api/turbines/{id}/state", a.handleUpdateTurbineState)
	mux.HandleFunc("POST /api/turbines", a.handleAddTurbine)
	mux.HandleFunc("POST /api/telemetry", a.handleIngest)
	mux.HandleFunc("GET /api/telemetry/{turbineID}/recent", a.handleRecentSamples)
	mux.HandleFunc("GET /api/alarms", a.handleListAlarms)
	mux.HandleFunc("POST /api/alarms/{id}/ack", a.handleAckAlarm)
	mux.HandleFunc("POST /api/alarms/{id}/resolve", a.handleResolveAlarm)
	mux.HandleFunc("GET /api/faults", a.handleListFaults)
	mux.HandleFunc("POST /api/faults", a.handleRaiseFault)
	mux.HandleFunc("POST /api/faults/{id}/resolve", a.handleResolveFault)
	mux.HandleFunc("POST /api/faults/{id}/ack", a.handleAckFault)
	mux.HandleFunc("POST /api/workorders", a.handleScheduleOrder)
	mux.HandleFunc("GET /api/workorders", a.handleListWorkorders)
	mux.HandleFunc("POST /api/workorders/{id}/assign", a.handleAssignOrder)
	mux.HandleFunc("GET /api/reports/{site}", a.handleReport)
	mux.HandleFunc("GET /api/audit", a.handleAuditHistory)
	mux.HandleFunc("GET /api/sites/stats", a.handleTurbineStats)
	mux.Handle("/", http.FileServer(http.Dir("web")))
	return mux
}

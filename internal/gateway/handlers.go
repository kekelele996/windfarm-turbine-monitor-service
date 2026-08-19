package gateway

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"windfarm-turbine-monitor-service/internal/alarm"
	"windfarm-turbine-monitor-service/internal/telemetry"
)

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("content-type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func (a *App) handleListTurbines(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, a.Turbines.List())
}

func (a *App) handleAddTurbine(w http.ResponseWriter, r *http.Request) {
	var req addTurbineRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	t := req.toTurbine()
	if t.ID == "" || t.Name == "" {
		writeError(w, http.StatusBadRequest, "id and name required")
		return
	}
	if err := a.Turbines.Put(t); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	a.Audit.Publish("turbine", t.ID, "registered")
	writeJSON(w, http.StatusCreated, t)
}

func (a *App) handleIngest(w http.ResponseWriter, r *http.Request) {
	var req ingestRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	validator := telemetry.DefaultValidator()
	valid, errs := validator.ValidateBatch(telemetry.SampleBatch{Source: req.Source, Samples: req.Samples})
	normalizer := telemetry.Normalizer{}
	normalizer.NormalizeBatch(valid)
	ingested, err := a.Ingestor.Ingest(r.Context(), valid)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	for _, s := range valid {
		for _, ev := range a.Evaluator.Evaluate(s) {
			candidate := alarm.Alarm{
				TurbineID: s.TurbineID,
				RuleID:    ev.Rule.ID,
				Metric:    string(ev.Rule.Metric),
				Severity:  string(ev.Rule.Severity),
				Value:     ev.Value,
				Threshold: ev.Rule.Threshold,
			}
			if created, ok, _ := a.Alarms.Dispatch(candidate); ok {
				a.Notifier.Notify(created)
				a.Audit.Publish("alarm", s.TurbineID, created.RuleID)
			}
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"ingested": ingested, "rejected": len(errs)})
}

func (a *App) handleRecentSamples(w http.ResponseWriter, r *http.Request) {
	turbineID := r.PathValue("turbineID")
	n, _ := strconv.Atoi(r.URL.Query().Get("n"))
	if n <= 0 {
		n = 20
	}
	writeJSON(w, http.StatusOK, a.Samples.Recent(turbineID, n))
}

func (a *App) handleListAlarms(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, a.Alarms.ListOpen())
}

func (a *App) handleAckAlarm(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	updated, err := a.Alarms.Ack(id)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

func (a *App) handleResolveAlarm(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	updated, err := a.Alarms.Resolve(id)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

func (a *App) handleListFaults(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, a.Faults.Open())
}

func (a *App) handleRaiseFault(w http.ResponseWriter, r *http.Request) {
	var req struct {
		TurbineID string `json:"turbine_id"`
		Code      string `json:"code"`
		Severity  string `json:"severity"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	f, err := a.Faults.Raise(req.TurbineID, req.Code, req.Severity)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, f)
}

func (a *App) handleResolveFault(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	f, err := a.Faults.Resolve(id)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, f)
}

func (a *App) handleScheduleOrder(w http.ResponseWriter, r *http.Request) {
	var req struct {
		TurbineID string `json:"turbine_id"`
		Title     string `json:"title"`
		Priority  int    `json:"priority"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	wo := a.Orders.Schedule(req.TurbineID, req.Title, req.Priority)
	a.Audit.Publish("work_order", req.TurbineID, wo.ID)
	writeJSON(w, http.StatusCreated, wo)
}

func (a *App) handleReport(w http.ResponseWriter, r *http.Request) {
	site := r.PathValue("site")
	now := time.Now()
	rep := a.Reports.Build(site, now.Add(-24*time.Hour), now)
	writeJSON(w, http.StatusOK, rep)
}

func (a *App) handleSummary(w http.ResponseWriter, r *http.Request) {
	stats := a.Turbines.Stats()
	writeJSON(w, http.StatusOK, map[string]any{
		"site":        "",
		"turbines":    stats.Total,
		"active":      stats.Active,
		"open_alarms": len(a.Alarms.ListOpen()),
		"open_faults": len(a.Faults.Open()),
	})
}

// idFromPath extracts the trailing path segment for older routers.
func idFromPath(path, prefix string) string {
	return strings.TrimPrefix(path, prefix)
}

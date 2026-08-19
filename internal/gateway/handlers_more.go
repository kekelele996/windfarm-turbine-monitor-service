package gateway

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

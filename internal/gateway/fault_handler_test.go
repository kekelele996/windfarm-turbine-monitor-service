package gateway

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"windfarm-turbine-monitor-service/internal/fault"
)

func TestNotFoundMapsTo404(t *testing.T) {
	app := &App{Faults: fault.NewService(fault.NewStore())}
	router := NewRouter(app)
	req := httptest.NewRequest(http.MethodPost, "/api/faults/missing-fault/resolve", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("status %d want %d", rr.Code, http.StatusNotFound)
	}
}

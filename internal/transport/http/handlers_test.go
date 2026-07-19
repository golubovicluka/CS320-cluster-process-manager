package httptransport

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golubovicluka/CS320-PZ/internal/cluster"
	"github.com/golubovicluka/CS320-PZ/internal/engine"
)

func testHandler(t *testing.T) http.Handler {
	t.Helper()
	controller, err := cluster.New(cluster.DefaultConfig())
	if err != nil {
		t.Fatal(err)
	}
	simulationEngine, err := engine.New(controller, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	handler, err := NewHandler(controller, simulationEngine)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(simulationEngine.Close)
	return handler
}

func request(t *testing.T, handler http.Handler, method, path string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var data []byte
	if body != nil {
		var err error
		data, err = json.Marshal(body)
		if err != nil {
			t.Fatal(err)
		}
	}
	req := httptest.NewRequest(method, path, bytes.NewReader(data))
	req.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, req)
	return recorder
}

func TestNodeProcessAndSimulationFlow(t *testing.T) {
	handler := testHandler(t)
	node := map[string]any{"id": "n1", "name": "worker", "cpuCapacity": 2, "memoryCapacityMB": 1024}
	if response := request(t, handler, http.MethodPost, "/api/v1/nodes", node); response.Code != http.StatusCreated {
		t.Fatalf("create node: status=%d body=%s", response.Code, response.Body.String())
	}
	if response := request(t, handler, http.MethodPost, "/api/v1/nodes", node); response.Code != http.StatusConflict {
		t.Fatalf("duplicate node: status=%d body=%s", response.Code, response.Body.String())
	}
	process := map[string]any{
		"id": "p1", "name": "job", "cpuRequest": 1, "memoryRequestMB": 128,
		"totalTicks": 2, "priority": 5, "restartPolicy": "NEVER", "maxRestarts": 0,
	}
	if response := request(t, handler, http.MethodPost, "/api/v1/processes", process); response.Code != http.StatusCreated {
		t.Fatalf("create process: status=%d body=%s", response.Code, response.Body.String())
	}
	response := request(t, handler, http.MethodPost, "/api/v1/simulation/step", map[string]int{"ticks": 2})
	if response.Code != http.StatusOK {
		t.Fatalf("step: status=%d body=%s", response.Code, response.Body.String())
	}
	response = request(t, handler, http.MethodGet, "/api/v1/metrics", nil)
	if response.Code != http.StatusOK {
		t.Fatalf("metrics: status=%d body=%s", response.Code, response.Body.String())
	}
	var report struct {
		Terminated int `json:"terminatedProcesses"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &report); err != nil {
		t.Fatal(err)
	}
	if report.Terminated != 1 {
		t.Fatalf("got %d terminated processes, want 1", report.Terminated)
	}
}

func TestErrorsAndExports(t *testing.T) {
	handler := testHandler(t)
	response := request(t, handler, http.MethodPost, "/api/v1/nodes", map[string]any{"id": "bad"})
	if response.Code != http.StatusBadRequest {
		t.Fatalf("invalid node: status=%d body=%s", response.Code, response.Body.String())
	}
	response = request(t, handler, http.MethodGet, "/api/v1/processes/missing", nil)
	if response.Code != http.StatusNotFound {
		t.Fatalf("missing process: status=%d body=%s", response.Code, response.Body.String())
	}
	response = request(t, handler, http.MethodGet, "/api/v1/reports/export?format=csv", nil)
	if response.Code != http.StatusOK || response.Header().Get("Content-Type") != "text/csv; charset=utf-8" {
		t.Fatalf("CSV export: status=%d content-type=%s", response.Code, response.Header().Get("Content-Type"))
	}
}

func TestRejectsInvalidJSON(t *testing.T) {
	handler := testHandler(t)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/nodes", bytes.NewBufferString(`{"id":`))
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, req)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
}

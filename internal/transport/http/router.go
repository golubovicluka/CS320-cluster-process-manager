package httptransport

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/golubovicluka/CS320-PZ/internal/cluster"
	"github.com/golubovicluka/CS320-PZ/internal/domain"
	"github.com/golubovicluka/CS320-PZ/internal/engine"
	"github.com/golubovicluka/CS320-PZ/internal/metrics"
)

type API struct {
	controller *cluster.Controller
	engine     *engine.Engine
	metrics    *metrics.Collector
	mux        *http.ServeMux
}

func NewHandler(controller *cluster.Controller, simulationEngine *engine.Engine) (http.Handler, error) {
	if controller == nil || simulationEngine == nil {
		return nil, fmt.Errorf("%w: controller and engine are required", domain.ErrInvalidInput)
	}
	collector, err := metrics.NewCollector(controller)
	if err != nil {
		return nil, err
	}
	api := &API{controller: controller, engine: simulationEngine, metrics: collector, mux: http.NewServeMux()}
	api.routes()
	return api.withRecovery(api.mux), nil
}

func (a *API) routes() {
	a.mux.HandleFunc("GET /healthz", a.health)

	a.mux.HandleFunc("POST /api/v1/nodes", a.createNode)
	a.mux.HandleFunc("GET /api/v1/nodes", a.listNodes)
	a.mux.HandleFunc("GET /api/v1/nodes/{id}", a.getNode)
	a.mux.HandleFunc("DELETE /api/v1/nodes/{id}", a.deleteNode)
	a.mux.HandleFunc("PATCH /api/v1/nodes/{id}/status", a.changeNodeStatus)
	a.mux.HandleFunc("POST /api/v1/nodes/{id}/fail", a.failNode)
	a.mux.HandleFunc("POST /api/v1/nodes/{id}/recover", a.recoverNode)
	a.mux.HandleFunc("POST /api/v1/nodes/{id}/heartbeat", a.heartbeat)

	a.mux.HandleFunc("POST /api/v1/processes", a.createProcess)
	a.mux.HandleFunc("GET /api/v1/processes", a.listProcesses)
	a.mux.HandleFunc("GET /api/v1/processes/{id}", a.getProcess)
	a.mux.HandleFunc("POST /api/v1/processes/{id}/pause", a.pauseProcess)
	a.mux.HandleFunc("POST /api/v1/processes/{id}/resume", a.resumeProcess)
	a.mux.HandleFunc("POST /api/v1/processes/{id}/kill", a.killProcess)
	a.mux.HandleFunc("POST /api/v1/processes/{id}/fail", a.failProcess)

	a.mux.HandleFunc("POST /api/v1/simulation/start", a.startSimulation)
	a.mux.HandleFunc("POST /api/v1/simulation/pause", a.pauseSimulation)
	a.mux.HandleFunc("POST /api/v1/simulation/step", a.stepSimulation)
	a.mux.HandleFunc("POST /api/v1/simulation/reset", a.resetSimulation)
	a.mux.HandleFunc("GET /api/v1/simulation/status", a.simulationStatus)
	a.mux.HandleFunc("POST /api/v1/simulation/scenarios", a.loadScenario)

	a.mux.HandleFunc("GET /api/v1/scheduler", a.getScheduler)
	a.mux.HandleFunc("PUT /api/v1/scheduler", a.setScheduler)
	a.mux.HandleFunc("GET /api/v1/events", a.listEvents)
	a.mux.HandleFunc("GET /api/v1/metrics", a.getMetrics)
	a.mux.HandleFunc("GET /api/v1/reports/summary", a.getMetrics)
	a.mux.HandleFunc("GET /api/v1/reports/export", a.exportReport)
}

func (a *API) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (a *API) withRecovery(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if recovered := recover(); recovered != nil {
				writeError(w, fmt.Errorf("internal panic: %v", recovered))
			}
		}()
		next.ServeHTTP(w, r)
	})
}

func statusForError(err error) int {
	switch {
	case errors.Is(err, domain.ErrNodeNotFound), errors.Is(err, domain.ErrProcessNotFound):
		return http.StatusNotFound
	case errors.Is(err, domain.ErrDuplicateNode), errors.Is(err, domain.ErrDuplicateProcess), errors.Is(err, domain.ErrInvalidStateTransition):
		return http.StatusConflict
	case errors.Is(err, domain.ErrInsufficientResources), errors.Is(err, domain.ErrNoSchedulableNode), errors.Is(err, domain.ErrMaxRestartsExceeded):
		return http.StatusUnprocessableEntity
	case errors.Is(err, domain.ErrInvalidInput):
		return http.StatusBadRequest
	default:
		return http.StatusInternalServerError
	}
}

func (a *API) shutdown() {
	a.engine.Close()
}

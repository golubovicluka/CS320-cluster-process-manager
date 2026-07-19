package httptransport

import (
	"context"
	"net/http"

	"github.com/golubovicluka/CS320-PZ/internal/domain"
	"github.com/golubovicluka/CS320-PZ/internal/scheduler"
)

func (a *API) startSimulation(w http.ResponseWriter, _ *http.Request) {
	if err := a.engine.Start(context.Background()); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, a.controller.Snapshot())
}

func (a *API) pauseSimulation(w http.ResponseWriter, _ *http.Request) {
	a.engine.Pause()
	writeJSON(w, http.StatusOK, a.controller.Snapshot())
}

func (a *API) stepSimulation(w http.ResponseWriter, r *http.Request) {
	var request struct {
		Ticks int `json:"ticks"`
	}
	if err := decodeJSON(w, r, &request); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	if request.Ticks == 0 {
		request.Ticks = 1
	}
	if err := a.engine.RunSteps(request.Ticks); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, a.controller.Snapshot())
}

func (a *API) resetSimulation(w http.ResponseWriter, _ *http.Request) {
	a.engine.Reset()
	writeJSON(w, http.StatusOK, a.controller.Snapshot())
}

func (a *API) simulationStatus(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, a.controller.Snapshot())
}

func (a *API) loadScenario(w http.ResponseWriter, r *http.Request) {
	var scenario domain.Scenario
	if err := decodeJSON(w, r, &scenario); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	if err := a.engine.LoadScenario(scenario); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, a.controller.Snapshot())
}

func (a *API) getScheduler(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"name":      a.controller.SchedulerName(),
		"available": scheduler.Available(),
	})
}

func (a *API) setScheduler(w http.ResponseWriter, r *http.Request) {
	var request struct {
		Name string `json:"name"`
	}
	if err := decodeJSON(w, r, &request); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	if err := a.controller.SetScheduler(request.Name); err != nil {
		writeError(w, err)
		return
	}
	a.getScheduler(w, r)
}

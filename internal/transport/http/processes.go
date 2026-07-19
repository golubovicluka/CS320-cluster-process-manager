package httptransport

import (
	"net/http"

	"github.com/golubovicluka/CS320-PZ/internal/domain"
)

func (a *API) createProcess(w http.ResponseWriter, r *http.Request) {
	var definition domain.ProcessDefinition
	if err := decodeJSON(w, r, &definition); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	process, err := a.controller.SubmitProcess(definition)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, process)
}

func (a *API) listProcesses(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, a.controller.Processes())
}

func (a *API) getProcess(w http.ResponseWriter, r *http.Request) {
	process, err := a.controller.Process(r.PathValue("id"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, process)
}

func (a *API) pauseProcess(w http.ResponseWriter, r *http.Request) {
	a.processAction(w, r, a.controller.PauseProcess)
}

func (a *API) resumeProcess(w http.ResponseWriter, r *http.Request) {
	a.processAction(w, r, a.controller.ResumeProcess)
}

func (a *API) waitProcess(w http.ResponseWriter, r *http.Request) {
	a.processAction(w, r, a.controller.WaitProcess)
}

func (a *API) wakeProcess(w http.ResponseWriter, r *http.Request) {
	a.processAction(w, r, a.controller.WakeProcess)
}

func (a *API) killProcess(w http.ResponseWriter, r *http.Request) {
	a.processAction(w, r, a.controller.KillProcess)
}

func (a *API) failProcess(w http.ResponseWriter, r *http.Request) {
	process, err := a.controller.FailProcess(r.PathValue("id"), "process failed by API request")
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, process)
}

func (a *API) processAction(w http.ResponseWriter, r *http.Request, action func(string) (*domain.Process, error)) {
	process, err := action(r.PathValue("id"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, process)
}

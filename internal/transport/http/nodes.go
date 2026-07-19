package httptransport

import (
	"net/http"

	"github.com/golubovicluka/CS320-PZ/internal/domain"
)

func (a *API) createNode(w http.ResponseWriter, r *http.Request) {
	var definition domain.NodeDefinition
	if err := decodeJSON(w, r, &definition); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	node, err := a.controller.RegisterNode(definition)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, node)
}

func (a *API) listNodes(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, a.controller.Nodes())
}

func (a *API) getNode(w http.ResponseWriter, r *http.Request) {
	node, err := a.controller.Node(r.PathValue("id"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, node)
}

func (a *API) deleteNode(w http.ResponseWriter, r *http.Request) {
	if err := a.controller.RemoveNode(r.PathValue("id")); err != nil {
		writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *API) changeNodeStatus(w http.ResponseWriter, r *http.Request) {
	var request struct {
		Status domain.NodeStatus `json:"status"`
	}
	if err := decodeJSON(w, r, &request); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	node, err := a.controller.ChangeNodeStatus(r.PathValue("id"), request.Status)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, node)
}

func (a *API) failNode(w http.ResponseWriter, r *http.Request) {
	node, err := a.controller.FailNode(r.PathValue("id"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, node)
}

func (a *API) recoverNode(w http.ResponseWriter, r *http.Request) {
	node, err := a.controller.RecoverNode(r.PathValue("id"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, node)
}

func (a *API) heartbeat(w http.ResponseWriter, r *http.Request) {
	if err := a.controller.Heartbeat(r.PathValue("id")); err != nil {
		writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

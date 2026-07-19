package httptransport

import (
	"net/http"

	"github.com/golubovicluka/CS320-PZ/internal/metrics"
)

func (a *API) listEvents(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, a.controller.Events())
}

func (a *API) getMetrics(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, a.metrics.Current())
}

func (a *API) exportReport(w http.ResponseWriter, r *http.Request) {
	report := a.metrics.Current()
	switch r.URL.Query().Get("format") {
	case "", "json":
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.Header().Set("Content-Disposition", `attachment; filename="simulation-report.json"`)
		if err := metrics.WriteJSON(w, report); err != nil {
			writeError(w, err)
		}
	case "csv":
		w.Header().Set("Content-Type", "text/csv; charset=utf-8")
		w.Header().Set("Content-Disposition", `attachment; filename="simulation-report.csv"`)
		if err := metrics.WriteCSV(w, report); err != nil {
			writeError(w, err)
		}
	default:
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "format must be json or csv"})
	}
}

package httpapi

import (
	"net/http"

	"subsurface-survey-gate/internal/application"
)

func (s *Server) RunScan(w http.ResponseWriter, r *http.Request) {
	var cmd application.RunScan
	if err := decode(w, r, &cmd); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", err.Error(), w.Header().Get("X-Request-ID"))
		return
	}
	result, err := s.service.RunScan(r.Context(), r.PathValue("campaignID"), cmd)
	if err != nil {
		handleError(w, r, err)
		return
	}
	writeResult(w, result)
}

func (s *Server) SubmitRectification(w http.ResponseWriter, r *http.Request) {
	var cmd application.SubmitRectification
	if err := decode(w, r, &cmd); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", err.Error(), w.Header().Get("X-Request-ID"))
		return
	}
	result, err := s.service.SubmitRectification(r.Context(), r.PathValue("campaignID"), cmd)
	if err != nil {
		handleError(w, r, err)
		return
	}
	writeResult(w, result)
}

func (s *Server) CompareScans(w http.ResponseWriter, r *http.Request) {
	result, err := s.service.CompareScans(r.Context(), r.PathValue("campaignID"), r.URL.Query().Get("baseScanId"), r.URL.Query().Get("targetScanId"))
	if err != nil {
		handleError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

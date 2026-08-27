package httpapi

import (
	"net/http"

	"subsurface-survey-gate/internal/application"
)

func (s *Server) AddObservation(w http.ResponseWriter, r *http.Request) {
	var cmd application.AddObservation
	if err := decode(w, r, &cmd); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", err.Error(), w.Header().Get("X-Request-ID"))
		return
	}
	result, err := s.service.AddObservation(r.Context(), r.PathValue("campaignID"), cmd)
	if err != nil {
		handleError(w, r, err)
		return
	}
	writeResult(w, result)
}

func (s *Server) AddObservationBatch(w http.ResponseWriter, r *http.Request) {
	var cmd application.AddObservationBatch
	if err := decode(w, r, &cmd); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", err.Error(), w.Header().Get("X-Request-ID"))
		return
	}
	result, err := s.service.AddObservationBatch(r.Context(), r.PathValue("campaignID"), cmd)
	if err != nil {
		handleError(w, r, err)
		return
	}
	writeResult(w, result)
}

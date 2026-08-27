package httpapi

import (
	"net/http"

	"subsurface-survey-gate/internal/application"
)

func (s *Server) SubmitReview(w http.ResponseWriter, r *http.Request) {
	var cmd application.SubmitReview
	if err := decode(w, r, &cmd); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", err.Error(), w.Header().Get("X-Request-ID"))
		return
	}
	result, err := s.service.SubmitReview(r.Context(), r.PathValue("campaignID"), cmd)
	if err != nil {
		handleError(w, r, err)
		return
	}
	writeResult(w, result)
}

func (s *Server) DecideReview(w http.ResponseWriter, r *http.Request) {
	var cmd application.DecideReview
	if err := decode(w, r, &cmd); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", err.Error(), w.Header().Get("X-Request-ID"))
		return
	}
	result, err := s.service.DecideReview(r.Context(), r.PathValue("campaignID"), cmd)
	if err != nil {
		handleError(w, r, err)
		return
	}
	writeResult(w, result)
}

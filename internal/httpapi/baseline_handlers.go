package httpapi

import (
	"net/http"

	"subsurface-survey-gate/internal/application"
)

func (s *Server) AddControl(w http.ResponseWriter, r *http.Request) {
	var cmd application.AddControl
	if err := decode(w, r, &cmd); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", err.Error(), w.Header().Get("X-Request-ID"))
		return
	}
	result, err := s.service.AddControl(r.Context(), r.PathValue("campaignID"), cmd)
	if err != nil {
		handleError(w, r, err)
		return
	}
	writeResult(w, result)
}

func (s *Server) LockBaseline(w http.ResponseWriter, r *http.Request) {
	var cmd application.LockBaseline
	if err := decode(w, r, &cmd); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", err.Error(), w.Header().Get("X-Request-ID"))
		return
	}
	result, err := s.service.LockBaseline(r.Context(), r.PathValue("campaignID"), cmd)
	if err != nil {
		handleError(w, r, err)
		return
	}
	writeResult(w, result)
}

func (s *Server) AmendControl(w http.ResponseWriter, r *http.Request) {
	var cmd application.AmendControl
	if err := decode(w, r, &cmd); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", err.Error(), w.Header().Get("X-Request-ID"))
		return
	}
	result, err := s.service.AmendControl(r.Context(), r.PathValue("campaignID"), r.PathValue("controlID"), cmd)
	if err != nil {
		handleError(w, r, err)
		return
	}
	writeResult(w, result)
}

func (s *Server) BaselineReadiness(w http.ResponseWriter, r *http.Request) {
	result, err := s.service.BaselineReadiness(r.Context(), r.PathValue("campaignID"))
	if err != nil {
		handleError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

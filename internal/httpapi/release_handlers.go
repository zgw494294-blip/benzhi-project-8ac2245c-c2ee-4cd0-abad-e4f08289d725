package httpapi

import (
	"net/http"

	"subsurface-survey-gate/internal/application"
)

func (s *Server) FreezeCampaign(w http.ResponseWriter, r *http.Request) {
	var cmd application.FreezeCampaign
	if err := decode(w, r, &cmd); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", err.Error(), w.Header().Get("X-Request-ID"))
		return
	}
	result, err := s.service.Freeze(r.Context(), r.PathValue("campaignID"), cmd)
	if err != nil {
		handleError(w, r, err)
		return
	}
	writeResult(w, result)
}

func (s *Server) IssueCredential(w http.ResponseWriter, r *http.Request) {
	var cmd application.IssueCredential
	if err := decode(w, r, &cmd); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", err.Error(), w.Header().Get("X-Request-ID"))
		return
	}
	result, err := s.service.IssueCredential(r.Context(), r.PathValue("campaignID"), cmd)
	if err != nil {
		handleError(w, r, err)
		return
	}
	writeResult(w, result)
}

func (s *Server) VerifyCredential(w http.ResponseWriter, r *http.Request) {
	var cmd application.VerifyCredential
	if err := decode(w, r, &cmd); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", err.Error(), w.Header().Get("X-Request-ID"))
		return
	}
	verification, err := s.service.VerifyCredential(r.Context(), cmd.Credential)
	if err != nil {
		handleError(w, r, err)
		return
	}
	status := http.StatusOK
	if !verification.Valid {
		status = http.StatusUnprocessableEntity
	}
	writeJSON(w, status, verification)
}

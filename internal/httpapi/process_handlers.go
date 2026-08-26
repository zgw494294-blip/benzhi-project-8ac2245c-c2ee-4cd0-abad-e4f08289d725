package httpapi

import (
	"net/http"

	"cleanroom-release-go/internal/domain"
	"cleanroom-release-go/internal/workflow"
)

func (s *Server) RecordPlannedObservation(w http.ResponseWriter, r *http.Request) {
	s.recordObservation(w, r, false)
}
func (s *Server) RecordVerificationObservation(w http.ResponseWriter, r *http.Request) {
	s.recordObservation(w, r, true)
}

func (s *Server) recordObservation(w http.ResponseWriter, r *http.Request, verification bool) {
	id, err := requiredPath(r, "campaignID")
	if err != nil {
		writeError(w, err)
		return
	}
	var cmd workflow.RecordObservationCommand
	if !decode(w, r, &cmd) {
		return
	}
	var c *domain.MonitoringCampaign
	if verification {
		c, err = s.workflow.RecordVerificationObservation(r.Context(), id, cmd)
	} else {
		c, err = s.workflow.RecordPlannedObservation(r.Context(), id, cmd)
	}
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, campaignResponse(c))
}

func (s *Server) ConcludeInvestigation(w http.ResponseWriter, r *http.Request) {
	campaignID, err := requiredPath(r, "campaignID")
	if err != nil {
		writeError(w, err)
		return
	}
	investigationID, err := requiredPath(r, "investigationID")
	if err != nil {
		writeError(w, err)
		return
	}
	var cmd workflow.ConcludeInvestigationCommand
	if !decode(w, r, &cmd) {
		return
	}
	c, err := s.workflow.ConcludeInvestigation(r.Context(), campaignID, investigationID, cmd)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, campaignResponse(c))
}

func (s *Server) AddCorrectiveAction(w http.ResponseWriter, r *http.Request) {
	id, err := requiredPath(r, "campaignID")
	if err != nil {
		writeError(w, err)
		return
	}
	var cmd workflow.AddCorrectiveActionCommand
	if !decode(w, r, &cmd) {
		return
	}
	c, err := s.workflow.AddCorrectiveAction(r.Context(), id, cmd)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, campaignResponse(c))
}

func (s *Server) CompleteCorrectiveAction(w http.ResponseWriter, r *http.Request) {
	id, err := requiredPath(r, "campaignID")
	if err != nil {
		writeError(w, err)
		return
	}
	actionID, err := requiredPath(r, "actionID")
	if err != nil {
		writeError(w, err)
		return
	}
	var cmd workflow.CompleteCorrectiveActionCommand
	if !decode(w, r, &cmd) {
		return
	}
	c, err := s.workflow.CompleteCorrectiveAction(r.Context(), id, actionID, cmd)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, campaignResponse(c))
}

func (s *Server) BeginVerification(w http.ResponseWriter, r *http.Request) {
	id, err := requiredPath(r, "campaignID")
	if err != nil {
		writeError(w, err)
		return
	}
	var cmd workflow.BeginVerificationCommand
	if !decode(w, r, &cmd) {
		return
	}
	c, err := s.workflow.BeginVerification(r.Context(), id, cmd)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, campaignResponse(c))
}

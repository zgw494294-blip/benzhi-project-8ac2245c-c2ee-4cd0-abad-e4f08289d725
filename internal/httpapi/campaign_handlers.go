package httpapi

import (
	"net/http"
	"time"

	"cleanroom-release-go/internal/workflow"
)

func (s *Server) Health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "time": time.Now().UTC()})
}

func (s *Server) CreateCampaign(w http.ResponseWriter, r *http.Request) {
	var cmd workflow.CreateCampaignCommand
	if !decode(w, r, &cmd) {
		return
	}
	c, err := s.workflow.CreateCampaign(r.Context(), cmd)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, campaignResponse(c))
}

func (s *Server) GetCampaign(w http.ResponseWriter, r *http.Request) {
	id, err := requiredPath(r, "campaignID")
	if err != nil {
		writeError(w, err)
		return
	}
	c, err := s.workflow.GetCampaign(r.Context(), id)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, campaignResponse(c))
}

func (s *Server) GetAuditTrail(w http.ResponseWriter, r *http.Request) {
	id, err := requiredPath(r, "campaignID")
	if err != nil {
		writeError(w, err)
		return
	}
	c, err := s.workflow.GetCampaign(r.Context(), id)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"campaignId": c.ID, "version": c.Version, "auditTrail": c.AuditTrail})
}

func (s *Server) LockPlan(w http.ResponseWriter, r *http.Request) {
	id, err := requiredPath(r, "campaignID")
	if err != nil {
		writeError(w, err)
		return
	}
	var cmd workflow.LockPlanCommand
	if !decode(w, r, &cmd) {
		return
	}
	c, err := s.workflow.LockPlan(r.Context(), id, cmd)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, campaignResponse(c))
}

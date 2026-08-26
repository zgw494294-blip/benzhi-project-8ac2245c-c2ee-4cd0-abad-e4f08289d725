package httpapi

import (
	"net/http"

	"cleanroom-release-go/internal/workflow"
)

func (s *Server) ReviewCampaign(w http.ResponseWriter, r *http.Request) {
	id, err := requiredPath(r, "campaignID")
	if err != nil {
		writeError(w, err)
		return
	}
	var cmd workflow.ReviewCommand
	if !decode(w, r, &cmd) {
		return
	}
	c, err := s.workflow.Review(r.Context(), id, cmd)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, campaignResponse(c))
}

func (s *Server) IssueCredential(w http.ResponseWriter, r *http.Request) {
	id, err := requiredPath(r, "campaignID")
	if err != nil {
		writeError(w, err)
		return
	}
	var cmd workflow.IssueCredentialCommand
	if !decode(w, r, &cmd) {
		return
	}
	credential, err := s.workflow.IssueCredential(r.Context(), id, cmd)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"credential": credential})
}

func (s *Server) VerifyCredential(w http.ResponseWriter, r *http.Request) {
	id, err := requiredPath(r, "credentialID")
	if err != nil {
		writeError(w, err)
		return
	}
	result, err := s.workflow.VerifyCredential(r.Context(), id)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"verification": result})
}

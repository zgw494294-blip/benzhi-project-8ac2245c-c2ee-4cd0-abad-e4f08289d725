package httpapi

import (
	"net/http"

	"subsurface-survey-gate/internal/application"
)

func (s *Server) Health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) CreateCampaign(w http.ResponseWriter, r *http.Request) {
	var cmd application.CreateCampaign
	if err := decode(w, r, &cmd); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", err.Error(), w.Header().Get("X-Request-ID"))
		return
	}
	result, err := s.service.CreateCampaign(r.Context(), cmd)
	if err != nil {
		handleError(w, r, err)
		return
	}
	writeResult(w, result)
}

func (s *Server) GetCampaign(w http.ResponseWriter, r *http.Request) {
	c, err := s.service.Campaign(r.Context(), r.PathValue("campaignID"))
	if err != nil {
		handleError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, c)
}

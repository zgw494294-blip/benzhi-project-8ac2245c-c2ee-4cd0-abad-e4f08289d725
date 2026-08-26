package httpapi

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"cleanroom-release-go/internal/domain"
	"cleanroom-release-go/internal/workflow"
)

func optionalTime(r *http.Request, key string) (*time.Time, error) {
	value := strings.TrimSpace(r.URL.Query().Get(key))
	if value == "" {
		return nil, nil
	}
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return nil, domain.Validation("invalid_"+key, "%s 必须是 RFC3339 时间", key)
	}
	parsed = parsed.UTC()
	return &parsed, nil
}

func optionalInt(r *http.Request, key string) (int, error) {
	value := strings.TrimSpace(r.URL.Query().Get(key))
	if value == "" {
		return 0, nil
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0, domain.Validation("invalid_"+key, "%s 必须是整数", key)
	}
	return parsed, nil
}

func optionalBool(r *http.Request, key string) (*bool, error) {
	value := strings.TrimSpace(r.URL.Query().Get(key))
	if value == "" {
		return nil, nil
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return nil, domain.Validation("invalid_"+key, "%s 必须是 true 或 false", key)
	}
	return &parsed, nil
}

func ledgerQuery(r *http.Request) (domain.CampaignLedgerQuery, error) {
	query := domain.CampaignLedgerQuery{FacilityName: strings.TrimSpace(r.URL.Query().Get("facilityName")), Cursor: strings.TrimSpace(r.URL.Query().Get("cursor"))}
	if value := strings.TrimSpace(r.URL.Query().Get("status")); value != "" {
		status, err := domain.ParseCampaignStatus(value)
		if err != nil {
			return query, err
		}
		query.Status = &status
	}
	var err error
	if query.CreatedFrom, err = optionalTime(r, "createdFrom"); err != nil {
		return query, err
	}
	if query.CreatedTo, err = optionalTime(r, "createdTo"); err != nil {
		return query, err
	}
	if query.PageSize, err = optionalInt(r, "pageSize"); err != nil {
		return query, err
	}
	if err := query.Validate(); err != nil {
		return query, err
	}
	return query, nil
}

func (s *Server) ListCampaigns(w http.ResponseWriter, r *http.Request) {
	query, err := ledgerQuery(r)
	if err != nil {
		writeError(w, err)
		return
	}
	page, err := s.workflow.ListCampaigns(r.Context(), query)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, page)
}

func (s *Server) GetCampaignStatistics(w http.ResponseWriter, r *http.Request) {
	query, err := ledgerQuery(r)
	if err != nil {
		writeError(w, err)
		return
	}
	statistics, err := s.workflow.CampaignStatistics(r.Context(), query)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, statistics)
}

func (s *Server) GetSamplingProgress(w http.ResponseWriter, r *http.Request) {
	id, err := requiredPath(r, "campaignID")
	if err != nil {
		writeError(w, err)
		return
	}
	round, err := optionalInt(r, "round")
	if err != nil {
		writeError(w, err)
		return
	}
	result, err := s.workflow.SamplingProgress(r.Context(), id, domain.SamplingProgressFilter{RoundNumber: round, AreaName: strings.TrimSpace(r.URL.Query().Get("area")), Metric: strings.TrimSpace(r.URL.Query().Get("metric"))})
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) UpdateInvestigationDraft(w http.ResponseWriter, r *http.Request) {
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
	var cmd workflow.UpdateInvestigationDraftCommand
	if !decode(w, r, &cmd) {
		return
	}
	c, err := s.workflow.UpdateInvestigationDraft(r.Context(), campaignID, investigationID, cmd)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, campaignResponse(c))
}

func (s *Server) GetInvestigationPreflight(w http.ResponseWriter, r *http.Request) {
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
	result, err := s.workflow.InvestigationPreflight(r.Context(), campaignID, investigationID)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) ListCorrectiveActions(w http.ResponseWriter, r *http.Request) {
	id, err := requiredPath(r, "campaignID")
	if err != nil {
		writeError(w, err)
		return
	}
	completed, err := optionalBool(r, "completed")
	if err != nil {
		writeError(w, err)
		return
	}
	overdue, err := optionalBool(r, "overdue")
	if err != nil {
		writeError(w, err)
		return
	}
	if status := strings.TrimSpace(r.URL.Query().Get("status")); status != "" {
		value := false
		switch status {
		case "completed":
			value = true
		case "outstanding":
		default:
			writeError(w, domain.Validation("invalid_action_status", "status 必须是 completed 或 outstanding"))
			return
		}
		if completed != nil && *completed != value {
			writeError(w, domain.Validation("conflicting_action_status", "completed 与 status 筛选条件冲突"))
			return
		}
		completed = &value
	}
	result, err := s.workflow.ListCorrectiveActions(r.Context(), id, domain.CorrectiveActionFilter{InvestigationID: strings.TrimSpace(r.URL.Query().Get("investigationId")), Owner: strings.TrimSpace(r.URL.Query().Get("owner")), Completed: completed, Overdue: overdue})
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) BatchAddCorrectiveActions(w http.ResponseWriter, r *http.Request) {
	id, err := requiredPath(r, "campaignID")
	if err != nil {
		writeError(w, err)
		return
	}
	var cmd workflow.BatchAddCorrectiveActionsCommand
	if !decode(w, r, &cmd) {
		return
	}
	c, err := s.workflow.BatchAddCorrectiveActions(r.Context(), id, cmd)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, campaignResponse(c))
}

func (s *Server) BatchCompleteCorrectiveActions(w http.ResponseWriter, r *http.Request) {
	id, err := requiredPath(r, "campaignID")
	if err != nil {
		writeError(w, err)
		return
	}
	var cmd workflow.BatchCompleteCorrectiveActionsCommand
	if !decode(w, r, &cmd) {
		return
	}
	c, err := s.workflow.BatchCompleteCorrectiveActions(r.Context(), id, cmd)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, campaignResponse(c))
}

func (s *Server) GetVerificationPreflight(w http.ResponseWriter, r *http.Request) {
	id, err := requiredPath(r, "campaignID")
	if err != nil {
		writeError(w, err)
		return
	}
	result, err := s.workflow.VerificationPreflight(r.Context(), id)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) GetVerificationComparison(w http.ResponseWriter, r *http.Request) {
	id, err := requiredPath(r, "campaignID")
	if err != nil {
		writeError(w, err)
		return
	}
	fromRound, err := optionalInt(r, "fromRound")
	if err != nil {
		writeError(w, err)
		return
	}
	toRound, err := optionalInt(r, "toRound")
	if err != nil {
		writeError(w, err)
		return
	}
	result, err := s.workflow.VerificationComparison(r.Context(), id, fromRound, toRound)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

package httpapi

import (
	"net/http"
	"strconv"

	"subsurface-survey-gate/internal/application"
	"subsurface-survey-gate/internal/domain"
)

func (s *Server) Issues(w http.ResponseWriter, r *http.Request) {
	values := r.URL.Query()
	pageSize := 50
	if raw := values.Get("pageSize"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil {
			handleError(w, r, domain.QueryError("pageSize", "必须为整数"))
			return
		}
		pageSize = parsed
	}
	query := application.IssueQuery{Filter: domain.IssueFilter{Status: domain.IssueStatus(values.Get("status")), Severity: domain.IssueSeverity(values.Get("severity")), RuleCode: values.Get("ruleCode"), ObjectRef: values.Get("objectRef")}, PageSize: pageSize, Cursor: values.Get("cursor")}
	result, err := s.service.Issues(r.Context(), r.PathValue("campaignID"), query)
	if err != nil {
		handleError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) AuditTimeline(w http.ResponseWriter, r *http.Request) {
	values := r.URL.Query()
	limit := 50
	if raw := values.Get("limit"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil {
			handleError(w, r, domain.QueryError("limit", "必须为整数"))
			return
		}
		limit = parsed
	}
	afterSequence := int64(0)
	if raw := values.Get("afterSequence"); raw != "" {
		parsed, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			handleError(w, r, domain.QueryError("afterSequence", "必须为整数"))
			return
		}
		afterSequence = parsed
	}
	result, err := s.service.AuditTimeline(r.Context(), r.PathValue("campaignID"), application.AuditQuery{EventType: values.Get("eventType"), Actor: values.Get("actor"), AfterSequence: afterSequence, Limit: limit, Cursor: values.Get("cursor")})
	if err != nil {
		handleError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

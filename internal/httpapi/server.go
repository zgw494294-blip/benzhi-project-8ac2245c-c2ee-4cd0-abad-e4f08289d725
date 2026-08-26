package httpapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"cleanroom-release-go/internal/domain"
	"cleanroom-release-go/internal/workflow"
)

type Server struct {
	workflow *workflow.Service
	logger   *slog.Logger
	mux      *http.ServeMux
}

func New(service *workflow.Service, logger *slog.Logger) *Server {
	if logger == nil {
		logger = slog.Default()
	}
	s := &Server{workflow: service, logger: logger, mux: http.NewServeMux()}
	s.routes()
	return s
}

func (s *Server) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		s.mux.ServeHTTP(w, r)
	})
}

func (s *Server) routes() {
	s.mux.HandleFunc("GET /healthz", s.Health)
	s.mux.HandleFunc("GET /api/v1/campaigns", s.ListCampaigns)
	s.mux.HandleFunc("GET /api/v1/campaigns/statistics", s.GetCampaignStatistics)
	s.mux.HandleFunc("POST /api/v1/campaigns", s.CreateCampaign)
	s.mux.HandleFunc("GET /api/v1/campaigns/{campaignID}", s.GetCampaign)
	s.mux.HandleFunc("GET /api/v1/campaigns/{campaignID}/audit", s.GetAuditTrail)
	s.mux.HandleFunc("POST /api/v1/campaigns/{campaignID}/plan/lock", s.LockPlan)
	s.mux.HandleFunc("POST /api/v1/campaigns/{campaignID}/observations/planned", s.RecordPlannedObservation)
	s.mux.HandleFunc("GET /api/v1/campaigns/{campaignID}/sampling/progress", s.GetSamplingProgress)
	s.mux.HandleFunc("PATCH /api/v1/campaigns/{campaignID}/investigations/{investigationID}/draft", s.UpdateInvestigationDraft)
	s.mux.HandleFunc("GET /api/v1/campaigns/{campaignID}/investigations/{investigationID}/preflight", s.GetInvestigationPreflight)
	s.mux.HandleFunc("POST /api/v1/campaigns/{campaignID}/investigations/{investigationID}/conclude", s.ConcludeInvestigation)
	s.mux.HandleFunc("POST /api/v1/campaigns/{campaignID}/corrective-actions", s.AddCorrectiveAction)
	s.mux.HandleFunc("GET /api/v1/campaigns/{campaignID}/corrective-actions", s.ListCorrectiveActions)
	s.mux.HandleFunc("POST /api/v1/campaigns/{campaignID}/corrective-actions/batch", s.BatchAddCorrectiveActions)
	s.mux.HandleFunc("POST /api/v1/campaigns/{campaignID}/corrective-actions/batch-complete", s.BatchCompleteCorrectiveActions)
	s.mux.HandleFunc("POST /api/v1/campaigns/{campaignID}/corrective-actions/{actionID}/complete", s.CompleteCorrectiveAction)
	s.mux.HandleFunc("POST /api/v1/campaigns/{campaignID}/verifications", s.BeginVerification)
	s.mux.HandleFunc("GET /api/v1/campaigns/{campaignID}/verifications/preflight", s.GetVerificationPreflight)
	s.mux.HandleFunc("GET /api/v1/campaigns/{campaignID}/verifications/comparison", s.GetVerificationComparison)
	s.mux.HandleFunc("POST /api/v1/campaigns/{campaignID}/observations/verification", s.RecordVerificationObservation)
	s.mux.HandleFunc("POST /api/v1/campaigns/{campaignID}/reviews", s.ReviewCampaign)
	s.mux.HandleFunc("POST /api/v1/campaigns/{campaignID}/credentials", s.IssueCredential)
	s.mux.HandleFunc("GET /api/v1/public/credentials/{credentialID}/verify", s.VerifyCredential)
}

type errorBody struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(value); err != nil {
		slog.Default().Error("编码 HTTP 响应失败", "error", err)
	}
}

func writeError(w http.ResponseWriter, err error) {
	status, code := http.StatusInternalServerError, "internal_error"
	var de *domain.Error
	if errors.As(err, &de) {
		code = de.Code
		switch de.Kind {
		case domain.KindValidation:
			status = http.StatusBadRequest
		case domain.KindNotFound:
			status = http.StatusNotFound
		case domain.KindConflict, domain.KindState, domain.KindFrozen:
			status = http.StatusConflict
		}
	}
	var body errorBody
	body.Error.Code, body.Error.Message = code, err.Error()
	writeJSON(w, status, body)
}

func decode(w http.ResponseWriter, r *http.Request, dst any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	d := json.NewDecoder(r.Body)
	d.DisallowUnknownFields()
	if err := d.Decode(dst); err != nil {
		writeError(w, domain.Validation("invalid_json", "JSON 请求无效: %v", err))
		return false
	}
	var extra any
	if err := d.Decode(&extra); err != io.EOF {
		writeError(w, domain.Validation("invalid_json", "请求只能包含一个 JSON 对象"))
		return false
	}
	return true
}

func requiredPath(r *http.Request, key string) (string, error) {
	value := strings.TrimSpace(r.PathValue(key))
	if value == "" {
		return "", domain.Validation("missing_path_parameter", "%s 不能为空", key)
	}
	return value, nil
}

func campaignResponse(c *domain.MonitoringCampaign) any {
	return struct {
		Campaign *domain.MonitoringCampaign `json:"campaign"`
		Version  int64                      `json:"version"`
	}{c, c.Version}
}

func methodName(r *http.Request) string { return fmt.Sprintf("%s %s", r.Method, r.URL.Path) }

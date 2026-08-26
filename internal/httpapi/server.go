package httpapi

import (
	"log/slog"
	"net/http"
	"time"

	"subsurface-survey-gate/internal/application"
)

type Server struct {
	service *application.Service
	logger  *slog.Logger
	mux     *http.ServeMux
}

func New(service *application.Service, logger *slog.Logger) http.Handler {
	if logger == nil {
		logger = slog.Default()
	}
	s := &Server{service: service, logger: logger, mux: http.NewServeMux()}
	s.routes()
	return s.middleware(s.mux)
}

func (s *Server) routes() {
	s.mux.HandleFunc("GET /healthz", s.Health)
	s.mux.HandleFunc("POST /api/v1/campaigns", s.CreateCampaign)
	s.mux.HandleFunc("GET /api/v1/campaigns/{campaignID}", s.GetCampaign)
	s.mux.HandleFunc("GET /api/v1/campaigns/{campaignID}/issues", s.Issues)
	s.mux.HandleFunc("GET /api/v1/campaigns/{campaignID}/audit-events", s.AuditTimeline)
	s.mux.HandleFunc("POST /api/v1/campaigns/{campaignID}/controls", s.AddControl)
	s.mux.HandleFunc("PATCH /api/v1/campaigns/{campaignID}/controls/{controlID}", s.AmendControl)
	s.mux.HandleFunc("GET /api/v1/campaigns/{campaignID}/baseline/readiness", s.BaselineReadiness)
	s.mux.HandleFunc("POST /api/v1/campaigns/{campaignID}/baseline/lock", s.LockBaseline)
	s.mux.HandleFunc("POST /api/v1/campaigns/{campaignID}/observations", s.AddObservation)
	s.mux.HandleFunc("POST /api/v1/campaigns/{campaignID}/observations/batch", s.AddObservationBatch)
	s.mux.HandleFunc("POST /api/v1/campaigns/{campaignID}/scans", s.RunScan)
	s.mux.HandleFunc("GET /api/v1/campaigns/{campaignID}/scans/compare", s.CompareScans)
	s.mux.HandleFunc("POST /api/v1/campaigns/{campaignID}/rectifications", s.SubmitRectification)
	s.mux.HandleFunc("POST /api/v1/campaigns/{campaignID}/review/submit", s.SubmitReview)
	s.mux.HandleFunc("POST /api/v1/campaigns/{campaignID}/review/decision", s.DecideReview)
	s.mux.HandleFunc("POST /api/v1/campaigns/{campaignID}/freeze", s.FreezeCampaign)
	s.mux.HandleFunc("POST /api/v1/campaigns/{campaignID}/credential", s.IssueCredential)
	s.mux.HandleFunc("POST /api/v1/credentials/verify", s.VerifyCredential)
}

type responseCapture struct {
	http.ResponseWriter
	status int
}

func (w *responseCapture) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

func (s *Server) middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		started := time.Now()
		requestID := r.Header.Get("X-Request-ID")
		if requestID == "" {
			requestID = time.Now().UTC().Format("20060102T150405.000000000")
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.Header().Set("X-Request-ID", requestID)
		w.Header().Set("X-Content-Type-Options", "nosniff")
		capture := &responseCapture{ResponseWriter: w, status: http.StatusOK}
		defer func() {
			if recovered := recover(); recovered != nil {
				writeError(capture, http.StatusInternalServerError, "internal_error", "服务内部错误", requestID)
			}
			s.logger.Info("http_access", "requestId", requestID, "method", r.Method, "path", r.URL.Path, "status", capture.status, "durationMs", time.Since(started).Milliseconds())
		}()
		next.ServeHTTP(capture, r)
	})
}

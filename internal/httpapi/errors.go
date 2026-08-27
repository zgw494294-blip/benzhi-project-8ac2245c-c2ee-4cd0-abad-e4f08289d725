package httpapi

import (
	"errors"
	"net/http"

	"subsurface-survey-gate/internal/domain"
)

type errorEnvelope struct {
	ErrorCode string `json:"errorCode"`
	Message   string `json:"message"`
	RequestID string `json:"requestId,omitempty"`
	Field     string `json:"field,omitempty"`
}

func handleError(w http.ResponseWriter, r *http.Request, err error) {
	requestID := w.Header().Get("X-Request-ID")
	var de *domain.Error
	if !errors.As(err, &de) {
		writeError(w, http.StatusInternalServerError, "internal_error", "服务内部错误", requestID)
		return
	}
	status := http.StatusBadRequest
	switch de.Kind {
	case domain.ErrorNotFound:
		status = http.StatusNotFound
	case domain.ErrorConflict, domain.ErrorVersion, domain.ErrorIdempotency:
		status = http.StatusConflict
	case domain.ErrorValidation:
		status = http.StatusUnprocessableEntity
	case domain.ErrorQuery:
		status = http.StatusBadRequest
	case domain.ErrorIntegrity:
		status = http.StatusInternalServerError
	}
	w.WriteHeader(status)
	_ = jsonNewEncoder(w).Encode(errorEnvelope{ErrorCode: string(de.Kind), Message: de.Message, RequestID: requestID, Field: de.Field})
}

func writeError(w http.ResponseWriter, status int, code, message, requestID string) {
	writeJSON(w, status, errorEnvelope{ErrorCode: code, Message: message, RequestID: requestID})
}

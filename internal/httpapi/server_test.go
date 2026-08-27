package httpapi

import (
	"bytes"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"subsurface-survey-gate/internal/application"
	"subsurface-survey-gate/internal/eventstore"
	"subsurface-survey-gate/internal/quality"
)

func TestStrictJSONAndIdempotentCreate(t *testing.T) {
	store, err := eventstore.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	handler := New(application.NewService(store, quality.NewScanner()), slog.New(slog.NewTextHandler(io.Discard, nil)))
	bad := httptest.NewRequest(http.MethodPost, "/api/v1/campaigns", strings.NewReader(`{"expectedVersion":0,"idempotencyKey":"k","name":"n","surveyArea":"a","coordinateReference":"c","specificationRevision":"r","unknown":1}`))
	badRecorder := httptest.NewRecorder()
	handler.ServeHTTP(badRecorder, bad)
	if badRecorder.Code != http.StatusBadRequest {
		t.Fatalf("未知字段状态码=%d", badRecorder.Code)
	}
	body := []byte(`{"expectedVersion":0,"idempotencyKey":"same","name":"n","surveyArea":"a","coordinateReference":"c","specificationRevision":"r","actor":"x"}`)
	first := httptest.NewRecorder()
	handler.ServeHTTP(first, httptest.NewRequest(http.MethodPost, "/api/v1/campaigns", bytes.NewReader(body)))
	second := httptest.NewRecorder()
	handler.ServeHTTP(second, httptest.NewRequest(http.MethodPost, "/api/v1/campaigns", bytes.NewReader(body)))
	if first.Code != http.StatusCreated || second.Code != http.StatusCreated || second.Header().Get("Idempotency-Replayed") != "true" {
		t.Fatalf("幂等创建失败: %d/%d", first.Code, second.Code)
	}
	if first.Body.String() != second.Body.String() {
		t.Fatal("幂等响应内容不一致")
	}
}

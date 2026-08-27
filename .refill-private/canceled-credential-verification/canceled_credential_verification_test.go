package canceledcredentialverification

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"subsurface-survey-gate/internal/application"
	"subsurface-survey-gate/internal/domain"
	"subsurface-survey-gate/internal/eventstore"
	"subsurface-survey-gate/internal/httpapi"
	"subsurface-survey-gate/internal/quality"
)

func TestCanceledCredentialVerificationPropagatesFailure(t *testing.T) {
	store, err := eventstore.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	service := application.NewService(store, quality.NewScanner())
	handler := httpapi.New(service, slog.New(slog.NewTextHandler(io.Discard, nil)))

	credential := domain.ReleaseCredential{
		ID:             "cred_canceled_lookup",
		CampaignID:     "cmp_canceled_lookup",
		FrozenVersion:  7,
		SnapshotDigest: "snapshot-digest",
		EventChainRoot: "event-chain-root",
		IssuedBy:       "tester",
		IssuedAt:       time.Date(2025, 1, 2, 3, 4, 5, 0, time.UTC),
	}
	credential.VerificationCode = domain.CredentialCode(credential)
	payload, err := json.Marshal(application.VerifyCredential{Credential: credential})
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/credentials/verify", bytes.NewReader(payload)).WithContext(ctx)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("取消错误被伪装为正常的凭据否定结果: status=%d body=%s", recorder.Code, recorder.Body.String())
	}
}

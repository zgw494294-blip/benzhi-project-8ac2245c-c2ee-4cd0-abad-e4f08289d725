package workflow

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"cleanroom-release-go/internal/domain"
)

type Service struct {
	repo   domain.CampaignRepository
	now    func() time.Time
	secret []byte
}

func New(repo domain.CampaignRepository, signingSecret string) *Service {
	if signingSecret == "" {
		signingSecret = "local-cleanroom-release-signing-key"
	}
	return &Service{repo: repo, now: time.Now, secret: []byte(signingSecret)}
}

func (s *Service) GetCampaign(ctx context.Context, id string) (*domain.MonitoringCampaign, error) {
	return s.repo.Get(ctx, strings.TrimSpace(id))
}

type WriteMeta struct {
	ExpectedVersion int64  `json:"expectedVersion"`
	IdempotencyKey  string `json:"idempotencyKey"`
	Actor           string `json:"actor"`
}

func mutation(campaignID, event string, meta WriteMeta, command any) (domain.Mutation, error) {
	if strings.TrimSpace(meta.IdempotencyKey) == "" {
		return domain.Mutation{}, domain.Validation("missing_idempotency_key", "idempotencyKey 不能为空")
	}
	if meta.ExpectedVersion < 0 {
		return domain.Mutation{}, domain.Validation("invalid_expected_version", "expectedVersion 不能小于 0")
	}
	if strings.TrimSpace(meta.Actor) == "" {
		return domain.Mutation{}, domain.Validation("missing_actor", "actor 不能为空")
	}
	b, err := json.Marshal(command)
	if err != nil {
		return domain.Mutation{}, fmt.Errorf("计算请求指纹: %w", err)
	}
	sum := sha256.Sum256(b)
	return domain.Mutation{CampaignID: campaignID, ExpectedVersion: meta.ExpectedVersion, IdempotencyKey: meta.IdempotencyKey, Fingerprint: hex.EncodeToString(sum[:]), EventType: event, Actor: meta.Actor}, nil
}

func (s *Service) signature(credential domain.ReleaseCredential) string {
	mac := hmac.New(sha256.New, s.secret)
	fmt.Fprintf(mac, "%s|%s|%d|%s|%s|%s", credential.ID, credential.CampaignID, credential.CampaignVersion, credential.SnapshotDigest, credential.IssuedBy, credential.IssuedAt.UTC().Format(time.RFC3339Nano))
	return hex.EncodeToString(mac.Sum(nil))
}

func ensureUnique(values ...string) error {
	seen := map[string]bool{}
	for _, value := range values {
		if strings.TrimSpace(value) == "" {
			return domain.Validation("missing_id", "业务对象 id 不能为空")
		}
		if seen[value] {
			return domain.Conflict("duplicate_id", "业务对象 id %s 已存在", value)
		}
		seen[value] = true
	}
	return nil
}

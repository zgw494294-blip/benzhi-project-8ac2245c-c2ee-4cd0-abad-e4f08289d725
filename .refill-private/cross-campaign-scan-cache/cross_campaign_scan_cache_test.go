package crosscampaignscancache

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"subsurface-survey-gate/internal/application"
	"subsurface-survey-gate/internal/domain"
	"subsurface-survey-gate/internal/quality"
)

type campaignStore struct {
	campaigns map[string]*domain.SurveyCampaign
}

func (s *campaignStore) Load(ctx context.Context, id string) (*domain.SurveyCampaign, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	campaign := domain.Clone(s.campaigns[id])
	if campaign == nil {
		return nil, domain.NotFound("批次")
	}
	return campaign, nil
}

func (s *campaignStore) LookupIdempotency(context.Context, string, string) (*domain.IdempotencyRecord, error) {
	return nil, nil
}

func (s *campaignStore) Commit(ctx context.Context, id string, expected int64, next *domain.SurveyCampaign, _ []domain.Event, _ domain.IdempotencyRecord) (*domain.SurveyCampaign, *domain.IdempotencyRecord, error) {
	if err := ctx.Err(); err != nil {
		return nil, nil, err
	}
	current := s.campaigns[id]
	if current == nil || current.Version != expected {
		return nil, nil, domain.VersionConflict()
	}
	s.campaigns[id] = domain.Clone(next)
	return domain.Clone(next), nil, nil
}

func (s *campaignStore) ChainRoot(context.Context, string) (string, error) {
	return "", fmt.Errorf("not used")
}

func (s *campaignStore) FindCredential(context.Context, string) (*domain.ReleaseCredential, error) {
	return nil, fmt.Errorf("not used")
}

func (s *campaignStore) AuditRecords(context.Context, string) ([]domain.AuditRecord, string, error) {
	return nil, "", fmt.Errorf("not used")
}

type campaignScanner struct {
	calls int
}

func (s *campaignScanner) Scan(c *domain.SurveyCampaign) quality.Result {
	s.calls++
	return quality.Result{
		RuleSetVersion: quality.RuleSetVersion,
		InputDigest:    domain.Digest(c.ID),
		Findings: []quality.Finding{{
			RuleCode:    "CAMPAIGN_REFERENCE",
			Severity:    domain.SeverityWarning,
			ObjectRef:   "campaign:" + c.ID,
			Description: "批次专属扫描结果",
		}},
	}
}

func (s *campaignScanner) BaselineReadiness(c *domain.SurveyCampaign) domain.BaselineReadiness {
	return domain.BaselineReadiness{CampaignID: c.ID, Version: c.Version, ReadyToLock: true}
}

func TestVersionOnlyScanCacheDoesNotLeakAcrossCampaigns(t *testing.T) {
	now := time.Date(2026, 8, 27, 9, 30, 0, 0, time.UTC)
	store := &campaignStore{campaigns: map[string]*domain.SurveyCampaign{
		"campaign-a": {ID: "campaign-a", State: domain.StateBaselineLocked, Version: 7, CreatedAt: now, UpdatedAt: now},
		"campaign-b": {ID: "campaign-b", State: domain.StateBaselineLocked, Version: 7, CreatedAt: now, UpdatedAt: now},
	}}
	scanner := &campaignScanner{}
	service := application.NewService(store, scanner)
	ctx := context.Background()

	if _, err := service.RunScan(ctx, "campaign-a", application.RunScan{Metadata: application.Metadata{ExpectedVersion: 7, IdempotencyKey: "scan-a"}, Actor: "复核员甲"}); err != nil {
		t.Fatalf("首个批次扫描失败: %v", err)
	}
	result, err := service.RunScan(ctx, "campaign-b", application.RunScan{Metadata: application.Metadata{ExpectedVersion: 7, IdempotencyKey: "scan-b"}, Actor: "复核员乙"})
	if err != nil {
		t.Fatalf("第二个批次扫描失败: %v", err)
	}
	var second domain.SurveyCampaign
	if err := json.Unmarshal(result.Body, &second); err != nil {
		t.Fatalf("解析第二个批次响应失败: %v", err)
	}
	if len(second.Issues) != 1 {
		t.Fatalf("第二个批次问题数=%d，期望 1", len(second.Issues))
	}
	want := "campaign:campaign-b"
	if got := second.Issues[0].ObjectRef; got != want {
		t.Fatalf("第二个批次复用了其他批次的扫描结果: objectRef=%q，期望 %q", got, want)
	}
	if scanner.calls != 2 {
		t.Fatalf("扫描器调用次数=%d，期望每个批次各调用一次", scanner.calls)
	}
}

package ledger

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"cleanroom-release-go/internal/domain"
)

type idempotencyRecord struct {
	Fingerprint string                     `json:"fingerprint"`
	CampaignID  string                     `json:"campaignId"`
	Version     int64                      `json:"version"`
	Result      *domain.MonitoringCampaign `json:"result"`
}

type projection struct {
	SchemaVersion int                                   `json:"schemaVersion"`
	LastSequence  int64                                 `json:"lastSequence"`
	LastDigest    string                                `json:"lastDigest"`
	Campaigns     map[string]*domain.MonitoringCampaign `json:"campaigns"`
	Idempotency   map[string]idempotencyRecord          `json:"idempotency"`
	Credentials   map[string]domain.ReleaseCredential   `json:"credentials"`
}

type Store struct {
	mu             sync.Mutex
	dir            string
	eventsPath     string
	projectionPath string
	state          projection
	now            func() time.Time
}

func Open(dir string) (*Store, error) {
	if dir == "" {
		return nil, fmt.Errorf("账本目录不能为空")
	}
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return nil, fmt.Errorf("创建账本目录: %w", err)
	}
	s := &Store{
		dir: dir, eventsPath: filepath.Join(dir, "events.jsonl"),
		projectionPath: filepath.Join(dir, "projection.json"), now: time.Now,
	}
	if err := s.recover(); err != nil {
		return nil, err
	}
	return s, nil
}

func emptyProjection() projection {
	return projection{SchemaVersion: schemaVersion, Campaigns: map[string]*domain.MonitoringCampaign{}, Idempotency: map[string]idempotencyRecord{}, Credentials: map[string]domain.ReleaseCredential{}}
}

func (s *Store) Get(_ context.Context, id string) (*domain.MonitoringCampaign, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	campaign, ok := s.state.Campaigns[id]
	if !ok {
		return nil, domain.NotFound("监测周期", id)
	}
	return domain.CloneCampaign(campaign)
}

func (s *Store) Create(_ context.Context, campaign *domain.MonitoringCampaign, meta domain.Mutation) (*domain.MonitoringCampaign, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if meta.ExpectedVersion != 0 {
		return nil, domain.Conflict("version_conflict", "新建周期 expectedVersion 必须为 0")
	}
	if prior, err := s.idempotentResult(meta); prior != nil || err != nil {
		return prior, err
	}
	if _, exists := s.state.Campaigns[campaign.ID]; exists {
		return nil, domain.Conflict("campaign_exists", "监测周期 %s 已存在", campaign.ID)
	}
	clone, err := domain.CloneCampaign(campaign)
	if err != nil {
		return nil, err
	}
	clone.Version = 1
	if err := s.commitCampaign(clone, meta); err != nil {
		return nil, err
	}
	return domain.CloneCampaign(clone)
}

func (s *Store) Update(_ context.Context, meta domain.Mutation, change func(*domain.MonitoringCampaign) error) (*domain.MonitoringCampaign, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if prior, err := s.idempotentResult(meta); prior != nil || err != nil {
		return prior, err
	}
	current, ok := s.state.Campaigns[meta.CampaignID]
	if !ok {
		return nil, domain.NotFound("监测周期", meta.CampaignID)
	}
	if current.Version != meta.ExpectedVersion {
		return nil, domain.Conflict("version_conflict", "expectedVersion=%d，当前版本=%d", meta.ExpectedVersion, current.Version)
	}
	clone, err := domain.CloneCampaign(current)
	if err != nil {
		return nil, err
	}
	// 变更函数看到的版本就是即将持久化的版本，冻结摘要因此无需依赖
	// 基础设施层之外的版本补偿逻辑。
	clone.Version++
	if err := change(clone); err != nil {
		return nil, err
	}
	if err := s.commitCampaign(clone, meta); err != nil {
		return nil, err
	}
	return domain.CloneCampaign(clone)
}

func (s *Store) idempotentResult(meta domain.Mutation) (*domain.MonitoringCampaign, error) {
	if meta.IdempotencyKey == "" || meta.Fingerprint == "" {
		return nil, domain.Validation("missing_concurrency_fields", "写命令必须包含 idempotencyKey、expectedVersion 和有效请求指纹")
	}
	key := idempotencyIndex(meta.CampaignID, meta.IdempotencyKey)
	record, ok := s.state.Idempotency[key]
	if !ok {
		return nil, nil
	}
	if record.Fingerprint != meta.Fingerprint {
		return nil, domain.Conflict("idempotency_conflict", "幂等键 %s 已用于不同请求", meta.IdempotencyKey)
	}
	campaign, ok := s.state.Campaigns[record.CampaignID]
	if !ok || campaign.Version < record.Version || record.Result == nil || record.Result.Version != record.Version {
		return nil, fmt.Errorf("幂等记录引用了无效投影")
	}
	return domain.CloneCampaign(record.Result)
}

func idempotencyIndex(campaignID, key string) string { return campaignID + "|" + key }

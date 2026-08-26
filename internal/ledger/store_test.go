package ledger

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"cleanroom-release-go/internal/domain"
)

func fixtureCampaign() *domain.MonitoringCampaign {
	return &domain.MonitoringCampaign{ID: "c1", FacilityName: "厂房", Status: domain.StatusDraft, Sites: []domain.ControlledSite{{ID: "s1", CampaignID: "c1", AreaName: "A", PointCode: "P1", CleanlinessGrade: "B", Metric: "settle_plate", Unit: "cfu/plate", AlertLimit: 2}}}
}
func fixtureMeta(version int64, key, event string) domain.Mutation {
	return domain.Mutation{CampaignID: "c1", ExpectedVersion: version, IdempotencyKey: key, Fingerprint: "fingerprint-" + key, EventType: event, Actor: "tester"}
}

func TestIdempotencyReturnsOriginalResult(t *testing.T) {
	store, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	first, err := store.Create(ctx, fixtureCampaign(), fixtureMeta(0, "create", "campaign.created"))
	if err != nil {
		t.Fatal(err)
	}
	second, err := store.Update(ctx, fixtureMeta(first.Version, "update", "campaign.updated"), func(c *domain.MonitoringCampaign) error { c.FacilityName = "新名称"; return nil })
	if err != nil {
		t.Fatal(err)
	}
	if second.Version != 2 {
		t.Fatalf("更新版本=%d", second.Version)
	}
	retry, err := store.Create(ctx, fixtureCampaign(), fixtureMeta(0, "create", "campaign.created"))
	if err != nil {
		t.Fatal(err)
	}
	if retry.Version != 1 || retry.FacilityName != "厂房" {
		t.Fatalf("幂等结果发生漂移: %+v", retry)
	}
	conflicting := fixtureMeta(0, "create", "campaign.created")
	conflicting.Fingerprint = "different"
	if _, err := store.Create(ctx, fixtureCampaign(), conflicting); err == nil {
		t.Fatal("相同幂等键的不同请求应冲突")
	}
}

func TestProjectionCorruptionRebuildsFromEventLedger(t *testing.T) {
	dir := t.TempDir()
	store, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Create(context.Background(), fixtureCampaign(), fixtureMeta(0, "create", "campaign.created")); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "projection.json"), []byte("not-json"), 0o640); err != nil {
		t.Fatal(err)
	}
	reopened, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	campaign, err := reopened.Get(context.Background(), "c1")
	if err != nil {
		t.Fatal(err)
	}
	if campaign.Version != 1 {
		t.Fatalf("重建版本=%d", campaign.Version)
	}
}

func TestTamperedEventChainIsRejected(t *testing.T) {
	dir := t.TempDir()
	store, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Create(context.Background(), fixtureCampaign(), fixtureMeta(0, "create", "campaign.created")); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "events.jsonl")
	payload, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	for i := range payload {
		if payload[i] == 'c' {
			payload[i] = 'x'
			break
		}
	}
	if err := os.WriteFile(path, payload, 0o640); err != nil {
		t.Fatal(err)
	}
	if _, err := Open(dir); err == nil {
		t.Fatal("被篡改的事件账本应拒绝启动")
	}
}

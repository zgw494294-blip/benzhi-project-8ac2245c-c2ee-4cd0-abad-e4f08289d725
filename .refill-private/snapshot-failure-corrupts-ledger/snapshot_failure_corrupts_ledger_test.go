package snapshotfailurecorruptsledger

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"subsurface-survey-gate/internal/domain"
	"subsurface-survey-gate/internal/eventstore"
)

func TestSnapshotFailureRetryKeepsLedgerRecoverable(t *testing.T) {
	dir := t.TempDir()
	store, err := eventstore.Open(dir)
	if err != nil {
		t.Fatal(err)
	}

	snapshotPath := filepath.Join(dir, "snapshots")
	if err := os.Remove(snapshotPath); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(snapshotPath, []byte("not a directory"), 0600); err != nil {
		t.Fatal(err)
	}

	now := time.Date(2025, 1, 2, 3, 4, 5, 0, time.UTC)
	campaign, err := domain.NewCampaign("cmp_snapshot_failure", "探测批次", "A 区", "CGCS2000", "r1", now)
	if err != nil {
		t.Fatal(err)
	}
	body, err := json.Marshal(campaign)
	if err != nil {
		t.Fatal(err)
	}
	event := domain.NewEvent(campaign, "evt_create", "campaign.created", "tester", now, campaign)
	idem := domain.IdempotencyRecord{
		CampaignID:  "__create__",
		Key:         "snapshot-retry",
		Fingerprint: "fixed-fingerprint",
		Status:      201,
		Response:    body,
	}

	if _, _, err := store.Commit(context.Background(), campaign.ID, 0, campaign, []domain.Event{event}, idem); err == nil {
		t.Fatal("快照路径失效时提交应返回错误")
	}
	if err := os.Remove(snapshotPath); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(snapshotPath, 0750); err != nil {
		t.Fatal(err)
	}

	if _, _, err := store.Commit(context.Background(), campaign.ID, 0, campaign, []domain.Event{event}, idem); err != nil {
		t.Fatalf("资源恢复后的同请求重试失败: %v", err)
	}
	if _, err := eventstore.Open(dir); err != nil {
		t.Fatalf("失败提交与重试生成了不可恢复的事件账本: %v", err)
	}
}

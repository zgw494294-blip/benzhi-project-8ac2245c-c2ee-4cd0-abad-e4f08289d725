package eventstore

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"subsurface-survey-gate/internal/domain"
)

func TestCommitRecoverAndIdempotency(t *testing.T) {
	dir := t.TempDir()
	store, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	c, _ := domain.NewCampaign("c1", "探测", "A区", "CGCS2000", "r1", now)
	body, _ := json.Marshal(c)
	idem := domain.IdempotencyRecord{CampaignID: "__create__", Key: "k1", Fingerprint: "fp", Status: 201, Response: body}
	event := domain.NewEvent(c, "e1", "campaign.created", "甲", now, c)
	if _, replay, err := store.Commit(context.Background(), c.ID, 0, c, []domain.Event{event}, idem); err != nil || replay != nil {
		t.Fatalf("首次提交失败: %v", err)
	}
	other := domain.Clone(c)
	other.ID = "other"
	if _, replay, err := store.Commit(context.Background(), "other", 0, other, []domain.Event{event}, idem); err != nil || replay == nil {
		t.Fatalf("创建幂等未重放: %v", err)
	}
	reopened, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	loaded, err := reopened.Load(context.Background(), "c1")
	if err != nil || loaded.Version != 1 {
		t.Fatalf("恢复失败: %v", err)
	}
	record, err := reopened.LookupIdempotency(context.Background(), "__create__", "k1")
	if err != nil || record == nil {
		t.Fatalf("幂等记录未恢复: %v", err)
	}
}

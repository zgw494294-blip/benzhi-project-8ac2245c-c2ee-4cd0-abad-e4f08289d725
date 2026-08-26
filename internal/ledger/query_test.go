package ledger

import (
	"context"
	"errors"
	"testing"
	"time"

	"cleanroom-release-go/internal/domain"
)

func TestCampaignLedgerPaginationStatisticsAndCursorValidation(t *testing.T) {
	store, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	base := time.Date(2026, 8, 27, 1, 0, 0, 0, time.UTC)
	values := []struct {
		id      string
		status  domain.CampaignStatus
		created time.Time
	}{
		{"draft-1", domain.StatusDraft, base.Add(-2 * time.Hour)},
		{"investigation-1", domain.StatusInvestigation, base.Add(-time.Hour)},
		{"frozen-1", domain.StatusFrozen, base},
	}
	for _, value := range values {
		campaign := &domain.MonitoringCampaign{ID: value.id, FacilityName: "一号厂房", Status: value.status, CreatedAt: value.created, Sites: []domain.ControlledSite{{ID: "site-" + value.id}}}
		_, err := store.Create(context.Background(), campaign, domain.Mutation{CampaignID: value.id, ExpectedVersion: 0, IdempotencyKey: "create-" + value.id, Fingerprint: value.id, EventType: "campaign.created", Actor: "test"})
		if err != nil {
			t.Fatal(err)
		}
	}
	query := domain.CampaignLedgerQuery{FacilityName: "一号", PageSize: 2}
	first, err := store.ListCampaigns(context.Background(), query)
	if err != nil {
		t.Fatal(err)
	}
	if len(first.Items) != 2 || first.Items[0].ID != "frozen-1" || first.NextCursor == "" {
		t.Fatalf("首屏异常: %+v", first)
	}
	query.Cursor = first.NextCursor
	second, err := store.ListCampaigns(context.Background(), query)
	if err != nil {
		t.Fatal(err)
	}
	if len(second.Items) != 1 || second.Items[0].ID != "draft-1" {
		t.Fatalf("次屏异常: %+v", second)
	}
	statistics, err := store.CampaignStatistics(context.Background(), domain.CampaignLedgerQuery{FacilityName: "一号"})
	if err != nil {
		t.Fatal(err)
	}
	if statistics.Total != 3 || statistics.ByStatus[domain.StatusFrozen] != 1 || statistics.PendingInvestigation != 1 || statistics.PendingCorrection != 0 {
		t.Fatalf("统计异常: %+v", statistics)
	}
	tampered := first.NextCursor[:len(first.NextCursor)/2] + "x" + first.NextCursor[len(first.NextCursor)/2+1:]
	query.Cursor = tampered
	_, err = store.ListCampaigns(context.Background(), query)
	var de *domain.Error
	if !errors.As(err, &de) || de.Code != "invalid_cursor" {
		t.Fatalf("篡改游标错误异常: %v", err)
	}
}

package ledger

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"sort"
	"strings"
	"time"

	"cleanroom-release-go/internal/domain"
)

type ledgerCursor struct {
	CreatedAt time.Time `json:"createdAt"`
	ID        string    `json:"id"`
	Filter    string    `json:"filter"`
	Checksum  string    `json:"checksum"`
}

func ledgerFilterDigest(q domain.CampaignLedgerQuery) string {
	var status string
	if q.Status != nil {
		status = string(*q.Status)
	}
	value := struct {
		FacilityName string     `json:"facilityName"`
		Status       string     `json:"status"`
		CreatedFrom  *time.Time `json:"createdFrom"`
		CreatedTo    *time.Time `json:"createdTo"`
	}{strings.TrimSpace(q.FacilityName), status, q.CreatedFrom, q.CreatedTo}
	b, _ := json.Marshal(value)
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

func cursorChecksum(createdAt time.Time, id, filter string) string {
	sum := sha256.Sum256([]byte(createdAt.UTC().Format(time.RFC3339Nano) + "|" + id + "|" + filter + "|cleanzone-ledger-cursor-v1"))
	return hex.EncodeToString(sum[:])
}

func encodeCursor(summary domain.CampaignLedgerSummary, filter string) (string, error) {
	cursor := ledgerCursor{CreatedAt: summary.CreatedAt, ID: summary.ID, Filter: filter}
	cursor.Checksum = cursorChecksum(cursor.CreatedAt, cursor.ID, cursor.Filter)
	b, err := json.Marshal(cursor)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func decodeCursor(value, filter string) (ledgerCursor, error) {
	var cursor ledgerCursor
	b, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil || json.Unmarshal(b, &cursor) != nil || cursor.ID == "" || cursor.CreatedAt.IsZero() ||
		cursor.Filter != filter || cursor.Checksum != cursorChecksum(cursor.CreatedAt, cursor.ID, cursor.Filter) {
		return cursor, domain.Validation("invalid_cursor", "游标无效或与当前筛选条件不匹配")
	}
	return cursor, nil
}

func matchesCampaign(c *domain.MonitoringCampaign, q domain.CampaignLedgerQuery) bool {
	if name := strings.TrimSpace(q.FacilityName); name != "" && !strings.Contains(strings.ToLower(c.FacilityName), strings.ToLower(name)) {
		return false
	}
	if q.Status != nil && c.Status != *q.Status {
		return false
	}
	if q.CreatedFrom != nil && c.CreatedAt.Before(*q.CreatedFrom) {
		return false
	}
	if q.CreatedTo != nil && c.CreatedAt.After(*q.CreatedTo) {
		return false
	}
	return true
}

func summarizeCampaign(c *domain.MonitoringCampaign) domain.CampaignLedgerSummary {
	last := c.CreatedAt
	for _, audit := range c.AuditTrail {
		if audit.CreatedAt.After(last) {
			last = audit.CreatedAt
		}
	}
	alerts := 0
	for _, observation := range c.Observations {
		if observation.Verdict == domain.VerdictAlert {
			alerts++
		}
	}
	return domain.CampaignLedgerSummary{ID: c.ID, FacilityName: c.FacilityName, Version: c.Version, Status: c.Status,
		CreatedAt: c.CreatedAt, SiteCount: len(c.Sites), AlertCount: alerts, LastActivityAt: last}
}

func (s *Store) matchingSummaries(q domain.CampaignLedgerQuery) ([]domain.CampaignLedgerSummary, error) {
	if err := q.Validate(); err != nil {
		return nil, err
	}
	result := make([]domain.CampaignLedgerSummary, 0, len(s.state.Campaigns))
	for id, campaign := range s.state.Campaigns {
		if campaign == nil {
			return nil, domain.Conflict("projection_missing", "监测周期 %s 的投影缺失", id)
		}
		if matchesCampaign(campaign, q) {
			result = append(result, summarizeCampaign(campaign))
		}
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].CreatedAt.Equal(result[j].CreatedAt) {
			return result[i].ID < result[j].ID
		}
		return result[i].CreatedAt.After(result[j].CreatedAt)
	})
	return result, nil
}

func (s *Store) ListCampaigns(_ context.Context, q domain.CampaignLedgerQuery) (domain.CampaignLedgerPage, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := q.Validate(); err != nil {
		return domain.CampaignLedgerPage{}, err
	}
	items, err := s.matchingSummaries(q)
	if err != nil {
		return domain.CampaignLedgerPage{}, err
	}
	filter := ledgerFilterDigest(q)
	start := 0
	if q.Cursor != "" {
		cursor, err := decodeCursor(q.Cursor, filter)
		if err != nil {
			return domain.CampaignLedgerPage{}, err
		}
		start = sort.Search(len(items), func(i int) bool {
			return items[i].CreatedAt.Before(cursor.CreatedAt) || (items[i].CreatedAt.Equal(cursor.CreatedAt) && items[i].ID > cursor.ID)
		})
	}
	end := start + q.PageSize
	if end > len(items) {
		end = len(items)
	}
	page := domain.CampaignLedgerPage{Items: append([]domain.CampaignLedgerSummary{}, items[start:end]...)}
	if end < len(items) && end > start {
		page.NextCursor, err = encodeCursor(items[end-1], filter)
		if err != nil {
			return domain.CampaignLedgerPage{}, err
		}
	}
	return page, nil
}

func (s *Store) CampaignStatistics(_ context.Context, q domain.CampaignLedgerQuery) (domain.CampaignStatusStatistics, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := q.Validate(); err != nil {
		return domain.CampaignStatusStatistics{}, err
	}
	items, err := s.matchingSummaries(q)
	if err != nil {
		return domain.CampaignStatusStatistics{}, err
	}
	result := domain.CampaignStatusStatistics{ByStatus: map[domain.CampaignStatus]int{}, Total: len(items)}
	for _, status := range []domain.CampaignStatus{domain.StatusDraft, domain.StatusPlanLocked, domain.StatusSampling, domain.StatusInvestigation, domain.StatusCorrection, domain.StatusVerification, domain.StatusVerificationPassed, domain.StatusFrozen} {
		result.ByStatus[status] = 0
	}
	for _, item := range items {
		result.ByStatus[item.Status]++
		switch item.Status {
		case domain.StatusInvestigation:
			result.PendingInvestigation++
		case domain.StatusCorrection:
			result.PendingCorrection++
		case domain.StatusVerification:
			result.PendingVerification++
		case domain.StatusVerificationPassed:
			result.PendingReview++
		}
	}
	return result, nil
}

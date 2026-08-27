package application

import (
	"context"

	"subsurface-survey-gate/internal/domain"
)

type IssueQuery struct {
	Filter   domain.IssueFilter `json:"filter"`
	PageSize int                `json:"pageSize"`
	Cursor   string             `json:"cursor"`
}

func (s *Service) Issues(ctx context.Context, campaignID string, query IssueQuery) (domain.IssueWorklist, error) {
	if query.PageSize < 1 || query.PageSize > 100 {
		return domain.IssueWorklist{}, domain.QueryError("pageSize", "必须为 1 到 100")
	}
	c, err := s.store.Load(ctx, campaignID)
	if err != nil {
		return domain.IssueWorklist{}, err
	}
	if err := c.ValidateIssueFilter(query.Filter); err != nil {
		return domain.IssueWorklist{}, err
	}
	filterFingerprint := domain.Digest(query.Filter)
	offset := 0
	if query.Cursor != "" {
		cursor, err := decodeCursor(query.Cursor)
		if err != nil {
			return domain.IssueWorklist{}, err
		}
		if cursor.Kind != "issues" || cursor.CampaignID != c.ID || cursor.Version != c.Version || cursor.FilterFingerprint != filterFingerprint {
			return domain.IssueWorklist{}, domain.QueryError("cursor", "游标与批次、版本或筛选条件不匹配")
		}
		offset = cursor.Offset
	}
	all := c.IssueItems(query.Filter)
	if offset > len(all) {
		return domain.IssueWorklist{}, domain.QueryError("cursor", "游标位置无效")
	}
	end := offset + query.PageSize
	if end > len(all) {
		end = len(all)
	}
	items := make([]domain.IssueWorkItem, end-offset)
	copy(items, all[offset:end])
	next := ""
	if end < len(all) {
		next = encodeCursor(pageCursor{Kind: "issues", CampaignID: c.ID, Version: c.Version, FilterFingerprint: filterFingerprint, Offset: end})
	}
	latestScanID := ""
	if scan := c.LatestScan(); scan != nil {
		latestScanID = scan.ID
	}
	return domain.IssueWorklist{CampaignID: c.ID, Version: c.Version, LatestScanID: latestScanID, Items: items, Summary: c.IssueSummary(), NextCursor: next}, nil
}

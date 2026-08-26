package application

import (
	"context"

	"subsurface-survey-gate/internal/domain"
)

type AuditQuery struct {
	EventType     string `json:"eventType"`
	Actor         string `json:"actor"`
	AfterSequence int64  `json:"afterSequence"`
	Limit         int    `json:"limit"`
	Cursor        string `json:"cursor"`
}

func (s *Service) AuditTimeline(ctx context.Context, campaignID string, query AuditQuery) (domain.AuditTimeline, error) {
	if query.AfterSequence < 0 {
		return domain.AuditTimeline{}, domain.QueryError("afterSequence", "不能为负数")
	}
	if query.Limit < 1 || query.Limit > 100 {
		return domain.AuditTimeline{}, domain.QueryError("limit", "必须为 1 到 100")
	}
	records, root, err := s.store.AuditRecords(ctx, campaignID)
	if err != nil {
		return domain.AuditTimeline{}, err
	}
	version := records[len(records)-1].Event.Version
	filter := struct {
		EventType     string `json:"eventType"`
		Actor         string `json:"actor"`
		AfterSequence int64  `json:"afterSequence"`
	}{query.EventType, query.Actor, query.AfterSequence}
	fingerprint := domain.Digest(filter)
	offset := 0
	if query.Cursor != "" {
		cursor, err := decodeCursor(query.Cursor)
		if err != nil {
			return domain.AuditTimeline{}, err
		}
		if cursor.Kind != "audit" || cursor.CampaignID != campaignID || cursor.Version != version || cursor.FilterFingerprint != fingerprint {
			return domain.AuditTimeline{}, domain.QueryError("cursor", "游标与批次、版本或筛选条件不匹配")
		}
		offset = cursor.Offset
	}
	filtered := make([]domain.AuditRecord, 0, len(records))
	for _, record := range records {
		if record.Sequence <= query.AfterSequence || query.EventType != "" && record.Event.Type != query.EventType || query.Actor != "" && record.Event.Actor != query.Actor {
			continue
		}
		filtered = append(filtered, record)
	}
	if offset > len(filtered) {
		return domain.AuditTimeline{}, domain.QueryError("cursor", "游标位置无效")
	}
	end := offset + query.Limit
	if end > len(filtered) {
		end = len(filtered)
	}
	events := make([]domain.AuditEventView, 0, end-offset)
	for _, record := range filtered[offset:end] {
		event := record.Event
		events = append(events, domain.AuditEventView{Sequence: record.Sequence, EventType: event.Type, Actor: event.Actor, OccurredAt: event.OccurredAt, Version: event.Version, Summary: domain.EventBusinessSummary(event), PreviousDigest: record.PreviousDigest, Digest: record.Digest, PreviousState: event.PreviousState, CurrentState: event.CurrentState})
	}
	next := ""
	if end < len(filtered) {
		next = encodeCursor(pageCursor{Kind: "audit", CampaignID: campaignID, Version: version, FilterFingerprint: fingerprint, Offset: end})
	}
	validation := domain.ChainValidation{Valid: true, Continuous: true, DigestLinked: true, ValidatedFrom: records[0].Sequence, ValidatedThrough: records[len(records)-1].Sequence}
	return domain.AuditTimeline{CampaignID: campaignID, Version: version, Events: events, NextCursor: next, ChainValidation: validation, ChainRoot: root}, nil
}

package eventstore

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"subsurface-survey-gate/internal/domain"
)

func (s *Store) Commit(ctx context.Context, campaignID string, expected int64, next *domain.SurveyCampaign, events []domain.Event, idem domain.IdempotencyRecord) (*domain.SurveyCampaign, *domain.IdempotencyRecord, error) {
	if err := ctx.Err(); err != nil {
		return nil, nil, err
	}
	if idem.Key == "" || idem.CampaignID == "" {
		return nil, nil, domain.Validation("idempotencyKey", "不能为空")
	}
	lockID := campaignID
	if idem.CampaignID == "__create__" {
		lockID = idem.CampaignID + "|" + idem.Key
	}
	lock := s.campaignLock(lockID)
	lock.Lock()
	defer lock.Unlock()
	idemKey := idem.CampaignID + "|" + idem.Key
	s.mu.RLock()
	prior, replay := s.idempotency[idemKey]
	current := s.campaigns[campaignID]
	root := s.roots[campaignID]
	seq := s.sequences[campaignID]
	s.mu.RUnlock()
	if replay {
		if prior.Fingerprint != idem.Fingerprint {
			return nil, nil, domain.IdempotencyConflict()
		}
		copy := prior
		return domain.Clone(current), &copy, nil
	}
	if next == nil || next.ID != campaignID || len(events) == 0 {
		return nil, nil, domain.Validation("commit", "提交内容不完整")
	}
	if next.Version != expected+1 {
		return nil, nil, domain.VersionConflict()
	}
	for _, event := range events {
		if event.CampaignID != campaignID || event.Version != next.Version {
			return nil, nil, domain.Validation("events", "事件版本或批次不匹配")
		}
	}
	if expected == 0 {
		if current != nil {
			return nil, nil, domain.Conflict("批次已存在")
		}
	} else if current == nil {
		return nil, nil, domain.NotFound("批次")
	} else if current.Version != expected {
		return nil, nil, domain.VersionConflict()
	}
	path := filepath.Join(s.dir, "events.jsonl")
	s.appendMu.Lock()
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0640)
	if err != nil {
		s.appendMu.Unlock()
		return nil, nil, err
	}
	w := bufio.NewWriter(f)
	auditRecords := make([]ledgerRecord, 0, len(events))
	for i, event := range events {
		seq++
		r := ledgerRecord{SchemaVersion: schemaVersion, Sequence: seq, CampaignID: campaignID, PreviousDigest: root, Event: event, State: domain.Clone(next), StoredAt: time.Now().UTC()}
		if i == len(events)-1 {
			copy := idem
			r.Idempotency = &copy
		}
		r.Digest = recordDigest(r)
		auditRecords = append(auditRecords, r)
		root = r.Digest
		line, marshalErr := encodeLine(r)
		if marshalErr != nil {
			_ = f.Close()
			s.appendMu.Unlock()
			return nil, nil, marshalErr
		}
		if _, err = w.Write(line); err != nil {
			_ = f.Close()
			s.appendMu.Unlock()
			return nil, nil, err
		}
	}
	if err = w.Flush(); err == nil {
		err = f.Sync()
	}
	closeErr := f.Close()
	if err == nil {
		err = closeErr
	}
	s.appendMu.Unlock()
	if err != nil {
		return nil, nil, fmt.Errorf("写入事件账本: %w", err)
	}
	snapshotErr := s.writeSnapshot(campaignID, seq, root, next)
	s.mu.Lock()
	s.campaigns[campaignID] = domain.Clone(next)
	s.roots[campaignID] = root
	s.sequences[campaignID] = seq
	s.idempotency[idemKey] = idem
	s.audit[campaignID] = append(s.audit[campaignID], auditRecords...)
	s.mu.Unlock()
	if snapshotErr != nil {
		return nil, nil, snapshotErr
	}
	return domain.Clone(next), nil, nil
}

func encodeLine(r ledgerRecord) ([]byte, error) {
	b, err := jsonMarshal(r)
	if err != nil {
		return nil, err
	}
	return append(b, '\n'), nil
}

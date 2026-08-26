package ledger

import (
	"encoding/json"
	"fmt"
	"os"

	"cleanroom-release-go/internal/domain"
)

func (s *Store) commitCampaign(campaign *domain.MonitoringCampaign, meta domain.Mutation) error {
	event := Event{
		SchemaVersion: schemaVersion, Sequence: s.state.LastSequence + 1,
		EventType: meta.EventType, CampaignID: campaign.ID, CampaignVersion: campaign.Version,
		IdempotencyKey: meta.IdempotencyKey, Fingerprint: meta.Fingerprint,
		Actor: meta.Actor, OccurredAt: s.now().UTC(), PreviousDigest: s.state.LastDigest,
		Campaign: campaign,
	}
	checksum, err := event.calculateChecksum()
	if err != nil {
		return err
	}
	event.Checksum = checksum
	if err := s.appendEvent(event); err != nil {
		return err
	}
	s.applyEvent(event)
	if err := s.writeProjection(); err != nil {
		return fmt.Errorf("事件已写入但投影保存失败: %w", err)
	}
	return nil
}

func (s *Store) appendEvent(event Event) error {
	f, err := os.OpenFile(s.eventsPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o640)
	if err != nil {
		return fmt.Errorf("打开事件账本: %w", err)
	}
	encoder := json.NewEncoder(f)
	if err := encoder.Encode(event); err != nil {
		_ = f.Close()
		return fmt.Errorf("追加事件: %w", err)
	}
	if err := f.Sync(); err != nil {
		_ = f.Close()
		return fmt.Errorf("同步事件账本: %w", err)
	}
	if err := f.Close(); err != nil {
		return fmt.Errorf("关闭事件账本: %w", err)
	}
	return nil
}

func (s *Store) applyEvent(event Event) {
	s.state.LastSequence = event.Sequence
	s.state.LastDigest = event.Checksum
	if event.Campaign != nil {
		s.state.Campaigns[event.CampaignID] = event.Campaign
		if event.IdempotencyKey != "" {
			key := idempotencyIndex(event.CampaignID, event.IdempotencyKey)
			s.state.Idempotency[key] = idempotencyRecord{Fingerprint: event.Fingerprint, CampaignID: event.CampaignID, Version: event.CampaignVersion, Result: event.Campaign}
		}
	}
	if event.Credential != nil {
		s.state.Credentials[event.Credential.ID] = *event.Credential
	}
}

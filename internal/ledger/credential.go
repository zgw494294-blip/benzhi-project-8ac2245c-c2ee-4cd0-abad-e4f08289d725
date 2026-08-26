package ledger

import (
	"context"
	"fmt"

	"cleanroom-release-go/internal/domain"
)

func (s *Store) SaveCredential(_ context.Context, credential domain.ReleaseCredential) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if existing, ok := s.state.Credentials[credential.ID]; ok {
		if existing.Signature == credential.Signature && existing.CampaignID == credential.CampaignID {
			return nil
		}
		return domain.Conflict("credential_exists", "凭据 %s 已存在且内容不同", credential.ID)
	}
	event := Event{
		SchemaVersion: schemaVersion, Sequence: s.state.LastSequence + 1,
		EventType: "credential.issued", CampaignID: credential.CampaignID,
		CampaignVersion: credential.CampaignVersion, Actor: credential.IssuedBy,
		OccurredAt: s.now().UTC(), PreviousDigest: s.state.LastDigest, Credential: &credential,
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
		return fmt.Errorf("凭据已写入但投影保存失败: %w", err)
	}
	return nil
}

func (s *Store) GetCredential(_ context.Context, id string) (domain.ReleaseCredential, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	credential, ok := s.state.Credentials[id]
	if !ok {
		return domain.ReleaseCredential{}, domain.NotFound("放行凭据", id)
	}
	return credential, nil
}

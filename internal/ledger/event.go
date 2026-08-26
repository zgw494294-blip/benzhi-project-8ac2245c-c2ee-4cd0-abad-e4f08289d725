package ledger

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"time"

	"cleanroom-release-go/internal/domain"
)

const schemaVersion = 1

type Event struct {
	SchemaVersion   int                        `json:"schemaVersion"`
	Sequence        int64                      `json:"sequence"`
	EventType       string                     `json:"eventType"`
	CampaignID      string                     `json:"campaignId,omitempty"`
	CampaignVersion int64                      `json:"campaignVersion,omitempty"`
	IdempotencyKey  string                     `json:"idempotencyKey,omitempty"`
	Fingerprint     string                     `json:"fingerprint,omitempty"`
	Actor           string                     `json:"actor,omitempty"`
	OccurredAt      time.Time                  `json:"occurredAt"`
	PreviousDigest  string                     `json:"previousDigest,omitempty"`
	Campaign        *domain.MonitoringCampaign `json:"campaign,omitempty"`
	Credential      *domain.ReleaseCredential  `json:"credential,omitempty"`
	Checksum        string                     `json:"checksum"`
}

type checksumEvent struct {
	SchemaVersion   int                        `json:"schemaVersion"`
	Sequence        int64                      `json:"sequence"`
	EventType       string                     `json:"eventType"`
	CampaignID      string                     `json:"campaignId,omitempty"`
	CampaignVersion int64                      `json:"campaignVersion,omitempty"`
	IdempotencyKey  string                     `json:"idempotencyKey,omitempty"`
	Fingerprint     string                     `json:"fingerprint,omitempty"`
	Actor           string                     `json:"actor,omitempty"`
	OccurredAt      time.Time                  `json:"occurredAt"`
	PreviousDigest  string                     `json:"previousDigest,omitempty"`
	Campaign        *domain.MonitoringCampaign `json:"campaign,omitempty"`
	Credential      *domain.ReleaseCredential  `json:"credential,omitempty"`
}

func (e Event) calculateChecksum() (string, error) {
	value := checksumEvent{
		SchemaVersion: e.SchemaVersion, Sequence: e.Sequence, EventType: e.EventType,
		CampaignID: e.CampaignID, CampaignVersion: e.CampaignVersion,
		IdempotencyKey: e.IdempotencyKey, Fingerprint: e.Fingerprint, Actor: e.Actor,
		OccurredAt: e.OccurredAt, PreviousDigest: e.PreviousDigest,
		Campaign: e.Campaign, Credential: e.Credential,
	}
	b, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:]), nil
}

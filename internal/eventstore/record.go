package eventstore

import (
	"encoding/json"
	"time"

	"subsurface-survey-gate/internal/domain"
)

const schemaVersion = 1

type ledgerRecord struct {
	SchemaVersion  int                       `json:"schemaVersion"`
	Sequence       int64                     `json:"sequence"`
	CampaignID     string                    `json:"campaignId"`
	PreviousDigest string                    `json:"previousDigest"`
	Digest         string                    `json:"digest"`
	Event          domain.Event              `json:"event"`
	State          *domain.SurveyCampaign    `json:"state"`
	Idempotency    *domain.IdempotencyRecord `json:"idempotency,omitempty"`
	StoredAt       time.Time                 `json:"storedAt"`
}

func recordDigest(r ledgerRecord) string {
	r.Digest = ""
	b, _ := json.Marshal(r)
	return domain.Digest(json.RawMessage(b))
}

type snapshotFile struct {
	SchemaVersion int                    `json:"schemaVersion"`
	CampaignID    string                 `json:"campaignId"`
	Sequence      int64                  `json:"sequence"`
	ChainRoot     string                 `json:"chainRoot"`
	State         *domain.SurveyCampaign `json:"state"`
	StateDigest   string                 `json:"stateDigest"`
	CreatedAt     time.Time              `json:"createdAt"`
}

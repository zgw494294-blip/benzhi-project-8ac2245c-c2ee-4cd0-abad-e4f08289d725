package domain

import (
	"context"
	"time"
)

type Mutation struct {
	CampaignID      string
	ExpectedVersion int64
	IdempotencyKey  string
	Fingerprint     string
	EventType       string
	Actor           string
}

type CampaignRepository interface {
	Create(ctx context.Context, campaign *MonitoringCampaign, meta Mutation) (*MonitoringCampaign, error)
	Update(ctx context.Context, meta Mutation, change func(*MonitoringCampaign) error) (*MonitoringCampaign, error)
	Get(ctx context.Context, id string) (*MonitoringCampaign, error)
	SaveCredential(ctx context.Context, credential ReleaseCredential) error
	GetCredential(ctx context.Context, id string) (ReleaseCredential, error)
	ListCampaigns(ctx context.Context, query CampaignLedgerQuery) (CampaignLedgerPage, error)
	CampaignStatistics(ctx context.Context, query CampaignLedgerQuery) (CampaignStatusStatistics, error)
}

type CampaignLedgerQuery struct {
	FacilityName string
	Status       *CampaignStatus
	CreatedFrom  *time.Time
	CreatedTo    *time.Time
	PageSize     int
	Cursor       string
}

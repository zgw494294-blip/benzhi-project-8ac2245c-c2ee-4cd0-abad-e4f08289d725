package application

import (
	"context"

	"subsurface-survey-gate/internal/domain"
	"subsurface-survey-gate/internal/quality"
)

type Store interface {
	Load(context.Context, string) (*domain.SurveyCampaign, error)
	LookupIdempotency(context.Context, string, string) (*domain.IdempotencyRecord, error)
	Commit(context.Context, string, int64, *domain.SurveyCampaign, []domain.Event, domain.IdempotencyRecord) (*domain.SurveyCampaign, *domain.IdempotencyRecord, error)
	ChainRoot(context.Context, string) (string, error)
	FindCredential(context.Context, string) (*domain.ReleaseCredential, error)
	AuditRecords(context.Context, string) ([]domain.AuditRecord, string, error)
}

type QualityScanner interface {
	Scan(*domain.SurveyCampaign) quality.Result
	BaselineReadiness(*domain.SurveyCampaign) domain.BaselineReadiness
}

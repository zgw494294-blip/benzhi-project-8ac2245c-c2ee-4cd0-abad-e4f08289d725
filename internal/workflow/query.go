package workflow

import (
	"context"

	"cleanroom-release-go/internal/assessment"
	"cleanroom-release-go/internal/domain"
)

func (s *Service) ListCampaigns(ctx context.Context, query domain.CampaignLedgerQuery) (domain.CampaignLedgerPage, error) {
	if err := query.Validate(); err != nil {
		return domain.CampaignLedgerPage{}, err
	}
	return s.repo.ListCampaigns(ctx, query)
}

func (s *Service) CampaignStatistics(ctx context.Context, query domain.CampaignLedgerQuery) (domain.CampaignStatusStatistics, error) {
	if err := query.Validate(); err != nil {
		return domain.CampaignStatusStatistics{}, err
	}
	query.Cursor = ""
	return s.repo.CampaignStatistics(ctx, query)
}

func (s *Service) SamplingProgress(ctx context.Context, campaignID string, filter domain.SamplingProgressFilter) (domain.SamplingProgress, error) {
	c, err := s.repo.Get(ctx, campaignID)
	if err != nil {
		return domain.SamplingProgress{}, err
	}
	return assessment.SamplingProgress(c, filter)
}

func (s *Service) InvestigationPreflight(ctx context.Context, campaignID, investigationID string) (domain.InvestigationClosePreflight, error) {
	c, err := s.repo.Get(ctx, campaignID)
	if err != nil {
		return domain.InvestigationClosePreflight{}, err
	}
	return assessment.InvestigationPreflight(c, investigationID)
}

func (s *Service) ListCorrectiveActions(ctx context.Context, campaignID string, filter domain.CorrectiveActionFilter) (domain.CorrectiveActionList, error) {
	c, err := s.repo.Get(ctx, campaignID)
	if err != nil {
		return domain.CorrectiveActionList{}, err
	}
	return assessment.CorrectiveActions(c, filter, s.now().UTC())
}

func (s *Service) VerificationPreflight(ctx context.Context, campaignID string) (domain.VerificationStartPreflight, error) {
	c, err := s.repo.Get(ctx, campaignID)
	if err != nil {
		return domain.VerificationStartPreflight{}, err
	}
	return assessment.VerificationPreflight(c), nil
}

func (s *Service) VerificationComparison(ctx context.Context, campaignID string, fromRound, toRound int) (domain.VerificationComparison, error) {
	c, err := s.repo.Get(ctx, campaignID)
	if err != nil {
		return domain.VerificationComparison{}, err
	}
	return assessment.VerificationRounds(c, fromRound, toRound)
}

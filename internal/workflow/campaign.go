package workflow

import (
	"context"
	"strings"

	"cleanroom-release-go/internal/domain"
)

type CreateCampaignCommand struct {
	WriteMeta
	ID           string                  `json:"id"`
	FacilityName string                  `json:"facilityName"`
	Sites        []domain.ControlledSite `json:"sites"`
}

func (s *Service) CreateCampaign(ctx context.Context, cmd CreateCampaignCommand) (*domain.MonitoringCampaign, error) {
	cmd.ID, cmd.FacilityName = strings.TrimSpace(cmd.ID), strings.TrimSpace(cmd.FacilityName)
	c := &domain.MonitoringCampaign{ID: cmd.ID, FacilityName: cmd.FacilityName, Status: domain.StatusDraft, CreatedAt: s.now().UTC(), Sites: cmd.Sites}
	if err := domain.ValidateNewCampaign(c); err != nil {
		return nil, err
	}
	meta, err := mutation(c.ID, "campaign.created", cmd.WriteMeta, cmd)
	if err != nil {
		return nil, err
	}
	c.AddAudit("campaign.created", cmd.Actor, "建立监测周期并登记受控点位", s.now().UTC())
	return s.repo.Create(ctx, c, meta)
}

type LockPlanCommand struct {
	WriteMeta
	Reviewer      string `json:"reviewer"`
	PlannedRounds int    `json:"plannedRounds"`
}

func (s *Service) LockPlan(ctx context.Context, campaignID string, cmd LockPlanCommand) (*domain.MonitoringCampaign, error) {
	meta, err := mutation(campaignID, "plan.locked", cmd.WriteMeta, cmd)
	if err != nil {
		return nil, err
	}
	return s.repo.Update(ctx, meta, func(c *domain.MonitoringCampaign) error {
		if err := c.EnsureMutable(); err != nil {
			return err
		}
		if err := c.RequireStatus(domain.StatusDraft); err != nil {
			return err
		}
		if err := domain.ValidateNewCampaign(c); err != nil {
			return err
		}
		if strings.TrimSpace(cmd.Reviewer) == "" || cmd.PlannedRounds < 1 {
			return domain.Validation("incomplete_plan", "reviewer 不能为空且 plannedRounds 必须大于 0")
		}
		c.PlanReviewer, c.PlannedRounds = strings.TrimSpace(cmd.Reviewer), cmd.PlannedRounds
		digest, err := c.ComputePlanDigest()
		if err != nil {
			return err
		}
		c.PlanDigest = digest
		if err := c.Transition(domain.StatusPlanLocked); err != nil {
			return err
		}
		c.AddAudit("plan.locked", cmd.Actor, "采样方案已锁定，摘要 "+digest, s.now().UTC())
		return nil
	})
}

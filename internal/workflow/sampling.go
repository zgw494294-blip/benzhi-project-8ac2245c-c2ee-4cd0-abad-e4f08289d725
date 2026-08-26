package workflow

import (
	"context"
	"fmt"
	"strings"
	"time"

	"cleanroom-release-go/internal/assessment"
	"cleanroom-release-go/internal/domain"
)

type RecordObservationCommand struct {
	WriteMeta
	ID            string    `json:"id"`
	SiteID        string    `json:"siteId"`
	RoundNumber   int       `json:"roundNumber"`
	ObservedValue float64   `json:"observedValue"`
	Unit          string    `json:"unit"`
	ObservedAt    time.Time `json:"observedAt,omitempty"`
}

func (s *Service) RecordPlannedObservation(ctx context.Context, campaignID string, cmd RecordObservationCommand) (*domain.MonitoringCampaign, error) {
	return s.recordObservation(ctx, campaignID, cmd, domain.RoundPlanned)
}

func (s *Service) RecordVerificationObservation(ctx context.Context, campaignID string, cmd RecordObservationCommand) (*domain.MonitoringCampaign, error) {
	return s.recordObservation(ctx, campaignID, cmd, domain.RoundVerification)
}

func (s *Service) recordObservation(ctx context.Context, campaignID string, cmd RecordObservationCommand, kind domain.RoundKind) (*domain.MonitoringCampaign, error) {
	event := "observation.planned_recorded"
	if kind == domain.RoundVerification {
		event = "observation.verification_recorded"
	}
	meta, err := mutation(campaignID, event, cmd.WriteMeta, cmd)
	if err != nil {
		return nil, err
	}
	return s.repo.Update(ctx, meta, func(c *domain.MonitoringCampaign) error {
		if err := c.EnsureMutable(); err != nil {
			return err
		}
		if kind == domain.RoundPlanned {
			if err := c.RequireStatus(domain.StatusPlanLocked, domain.StatusSampling); err != nil {
				return err
			}
			if cmd.RoundNumber < 1 || cmd.RoundNumber > c.PlannedRounds {
				return domain.Validation("invalid_round", "计划轮次必须在 1 到 %d 之间", c.PlannedRounds)
			}
		} else {
			if err := c.RequireStatus(domain.StatusVerification); err != nil {
				return err
			}
			if cmd.RoundNumber != c.VerificationRound {
				return domain.Validation("invalid_verification_round", "当前验证轮次为 %d", c.VerificationRound)
			}
		}
		if strings.TrimSpace(cmd.ID) == "" {
			return domain.Validation("missing_observation_id", "采样结果 id 不能为空")
		}
		for _, old := range c.Observations {
			if old.ID == cmd.ID {
				return domain.Conflict("observation_exists", "采样结果 %s 已存在", cmd.ID)
			}
			if old.SiteID == cmd.SiteID && old.RoundKind == kind && old.RoundNumber == cmd.RoundNumber {
				return domain.Conflict("observation_duplicate", "该监测点在本轮已有结果")
			}
		}
		site, err := c.Site(cmd.SiteID)
		if err != nil {
			return err
		}
		decision, err := assessment.AssessObservation(*site, cmd.ObservedValue, cmd.Unit)
		if err != nil {
			return err
		}
		at := cmd.ObservedAt.UTC()
		if cmd.ObservedAt.IsZero() {
			at = s.now().UTC()
		}
		observation := domain.SampleObservation{ID: cmd.ID, CampaignID: c.ID, SiteID: cmd.SiteID, RoundNumber: cmd.RoundNumber, RoundKind: kind, ObservedValue: cmd.ObservedValue, Unit: cmd.Unit, Verdict: decision.Verdict, Explanation: decision.Explanation, ObservedAt: at}
		c.Observations = append(c.Observations, observation)
		if decision.Verdict == domain.VerdictAlert {
			invID := "inv-" + cmd.ID
			c.Investigations = append(c.Investigations, domain.DeviationInvestigation{ID: invID, CampaignID: c.ID, ObservationID: cmd.ID, Status: domain.InvestigationOpen, OpenedAt: s.now().UTC()})
			if err := c.Transition(domain.StatusInvestigation); err != nil {
				return err
			}
			c.AddAudit("deviation.opened", cmd.Actor, fmt.Sprintf("采样 %s 超限并自动建立调查 %s", cmd.ID, invID), s.now().UTC())
			return nil
		}
		if kind == domain.RoundPlanned {
			if err := c.Transition(domain.StatusSampling); err != nil {
				return err
			}
			c.AddAudit("observation.recorded", cmd.Actor, decision.Explanation, s.now().UTC())
			return nil
		}
		result := assessment.AssessVerification(c, c.VerificationRound)
		if result.Complete && result.Passed {
			if err := c.Transition(domain.StatusVerificationPassed); err != nil {
				return err
			}
		}
		c.AddAudit("verification.observation_recorded", cmd.Actor, result.Reason, s.now().UTC())
		return nil
	})
}

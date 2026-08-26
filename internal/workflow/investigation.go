package workflow

import (
	"context"
	"fmt"
	"strings"

	"cleanroom-release-go/internal/assessment"
	"cleanroom-release-go/internal/domain"
)

type UpdateInvestigationDraftCommand struct {
	WriteMeta
	ImpactScope  *string   `json:"impactScope,omitempty"`
	Hypotheses   *[]string `json:"hypotheses,omitempty"`
	EvidenceRefs *[]string `json:"evidenceRefs,omitempty"`
}

func (s *Service) UpdateInvestigationDraft(ctx context.Context, campaignID, investigationID string, cmd UpdateInvestigationDraftCommand) (*domain.MonitoringCampaign, error) {
	meta, err := mutation(campaignID, "investigation.draft_updated", cmd.WriteMeta, cmd)
	if err != nil {
		return nil, err
	}
	return s.repo.Update(ctx, meta, func(c *domain.MonitoringCampaign) error {
		if err := c.EnsureMutable(); err != nil {
			return err
		}
		if err := c.RequireStatus(domain.StatusInvestigation); err != nil {
			return err
		}
		inv, err := c.Investigation(investigationID)
		if err != nil {
			return err
		}
		if inv.CampaignID != c.ID {
			return domain.NotFound("偏差调查", investigationID)
		}
		changed, err := domain.MergeInvestigationDraft(inv, domain.InvestigationDraftPatch{ImpactScope: cmd.ImpactScope, Hypotheses: cmd.Hypotheses, EvidenceRefs: cmd.EvidenceRefs})
		if err != nil {
			return err
		}
		c.AddAudit("investigation.draft_updated", cmd.Actor, fmt.Sprintf("调查 %s 更新字段: %s", investigationID, strings.Join(changed, ",")), s.now().UTC())
		return nil
	})
}

type ConcludeInvestigationCommand struct {
	WriteMeta
	ImpactScope  string   `json:"impactScope"`
	Hypotheses   []string `json:"hypotheses"`
	EvidenceRefs []string `json:"evidenceRefs"`
	RootCause    string   `json:"rootCause"`
}

func (s *Service) ConcludeInvestigation(ctx context.Context, campaignID, investigationID string, cmd ConcludeInvestigationCommand) (*domain.MonitoringCampaign, error) {
	meta, err := mutation(campaignID, "investigation.concluded", cmd.WriteMeta, cmd)
	if err != nil {
		return nil, err
	}
	return s.repo.Update(ctx, meta, func(c *domain.MonitoringCampaign) error {
		if err := c.EnsureMutable(); err != nil {
			return err
		}
		if err := c.RequireStatus(domain.StatusInvestigation); err != nil {
			return err
		}
		inv, err := c.Investigation(investigationID)
		if err != nil {
			return err
		}
		if inv.Status == domain.InvestigationConcluded {
			return domain.Conflict("investigation_closed", "调查 %s 已闭合", investigationID)
		}
		var patch domain.InvestigationDraftPatch
		if strings.TrimSpace(cmd.ImpactScope) != "" {
			value := cmd.ImpactScope
			patch.ImpactScope = &value
		}
		if len(cmd.Hypotheses) > 0 {
			values := cmd.Hypotheses
			patch.Hypotheses = &values
		}
		if len(cmd.EvidenceRefs) > 0 {
			values := cmd.EvidenceRefs
			patch.EvidenceRefs = &values
		}
		if patch.ImpactScope != nil || patch.Hypotheses != nil || patch.EvidenceRefs != nil {
			if _, err := domain.MergeInvestigationDraft(inv, patch); err != nil {
				return err
			}
		}
		inv.RootCause = strings.TrimSpace(cmd.RootCause)
		complete := assessment.InvestigationCompleteness(*inv)
		if !complete.Complete {
			return domain.Validation("incomplete_investigation", "调查缺少字段: %s", strings.Join(complete.Missing, ","))
		}
		now := s.now().UTC()
		inv.Status, inv.ConcludedAt = domain.InvestigationConcluded, &now
		if !c.HasOpenInvestigation() {
			if err := c.Transition(domain.StatusCorrection); err != nil {
				return err
			}
		}
		c.AddAudit("investigation.concluded", cmd.Actor, "调查 "+investigationID+" 已确认根因", now)
		return nil
	})
}

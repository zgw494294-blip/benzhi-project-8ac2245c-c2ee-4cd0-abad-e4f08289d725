package workflow

import (
	"context"
	"fmt"
	"strings"
	"time"

	"cleanroom-release-go/internal/assessment"
	"cleanroom-release-go/internal/domain"
)

type CorrectiveActionInput struct {
	ID              string    `json:"id"`
	InvestigationID string    `json:"investigationId"`
	Description     string    `json:"description"`
	Owner           string    `json:"owner"`
	DueAt           time.Time `json:"dueAt"`
}

type BatchAddCorrectiveActionsCommand struct {
	WriteMeta
	Actions []CorrectiveActionInput `json:"actions"`
}

func (s *Service) BatchAddCorrectiveActions(ctx context.Context, campaignID string, cmd BatchAddCorrectiveActionsCommand) (*domain.MonitoringCampaign, error) {
	if len(cmd.Actions) < 1 || len(cmd.Actions) > domain.MaxCorrectionBatchSize {
		return nil, domain.Validation("invalid_batch_size", "actions 数量必须在 1 到 %d 之间", domain.MaxCorrectionBatchSize)
	}
	meta, err := mutation(campaignID, "correction.batch_created", cmd.WriteMeta, cmd)
	if err != nil {
		return nil, err
	}
	return s.repo.Update(ctx, meta, func(c *domain.MonitoringCampaign) error {
		if err := c.EnsureMutable(); err != nil {
			return err
		}
		if err := c.RequireStatus(domain.StatusCorrection); err != nil {
			return err
		}
		now := s.now().UTC()
		existing := map[string]bool{}
		for _, action := range c.Actions {
			existing[action.ID] = true
		}
		batch := map[string]bool{}
		prepared := make([]domain.CorrectiveAction, 0, len(cmd.Actions))
		for i, input := range cmd.Actions {
			action := domain.CorrectiveAction{ID: strings.TrimSpace(input.ID), InvestigationID: strings.TrimSpace(input.InvestigationID), Description: strings.TrimSpace(input.Description), Owner: strings.TrimSpace(input.Owner), DueAt: input.DueAt.UTC()}
			if err := domain.ValidateCorrectiveAction(action, now, true); err != nil {
				return fmt.Errorf("actions[%d]: %w", i, err)
			}
			if batch[action.ID] {
				return domain.Validation("duplicate_action_id", "纠正措施编号 %s 在批次内重复", action.ID)
			}
			if existing[action.ID] {
				return domain.Conflict("action_exists", "纠正措施编号 %s 在周期内已存在", action.ID)
			}
			inv, err := c.Investigation(action.InvestigationID)
			if err != nil {
				return fmt.Errorf("actions[%d]: %w", i, err)
			}
			if inv.CampaignID != c.ID {
				return domain.NotFound("偏差调查", action.InvestigationID)
			}
			if inv.Status != domain.InvestigationConcluded || strings.TrimSpace(inv.RootCause) == "" {
				return domain.InvalidState("调查 %s 尚未确认根因", inv.ID)
			}
			batch[action.ID] = true
			prepared = append(prepared, action)
		}
		c.Actions = append(c.Actions, prepared...)
		c.AddAudit("correction.batch_created", cmd.Actor, fmt.Sprintf("批量登记 %d 项纠正措施", len(prepared)), now)
		return nil
	})
}

type CorrectiveActionCompletionInput struct {
	ID                 string `json:"id"`
	CompletionEvidence string `json:"completionEvidence"`
}

type BatchCompleteCorrectiveActionsCommand struct {
	WriteMeta
	Actions []CorrectiveActionCompletionInput `json:"actions"`
}

func (s *Service) BatchCompleteCorrectiveActions(ctx context.Context, campaignID string, cmd BatchCompleteCorrectiveActionsCommand) (*domain.MonitoringCampaign, error) {
	if len(cmd.Actions) < 1 || len(cmd.Actions) > domain.MaxCorrectionBatchSize {
		return nil, domain.Validation("invalid_batch_size", "actions 数量必须在 1 到 %d 之间", domain.MaxCorrectionBatchSize)
	}
	meta, err := mutation(campaignID, "correction.batch_completed", cmd.WriteMeta, cmd)
	if err != nil {
		return nil, err
	}
	return s.repo.Update(ctx, meta, func(c *domain.MonitoringCampaign) error {
		if err := c.EnsureMutable(); err != nil {
			return err
		}
		if err := c.RequireStatus(domain.StatusCorrection); err != nil {
			return err
		}
		seen := map[string]bool{}
		resolved := make([]*domain.CorrectiveAction, 0, len(cmd.Actions))
		evidence := make([]string, 0, len(cmd.Actions))
		for i, input := range cmd.Actions {
			id, proof := strings.TrimSpace(input.ID), strings.TrimSpace(input.CompletionEvidence)
			if id == "" {
				return domain.Validation("missing_action_id", "actions[%d].id 不能为空", i)
			}
			if seen[id] {
				return domain.Validation("duplicate_action_id", "批次内纠正措施编号 %s 重复", id)
			}
			if proof == "" {
				return domain.Validation("missing_completion_evidence", "actions[%d].completionEvidence 不能为空", i)
			}
			action, err := c.Action(id)
			if err != nil {
				return err
			}
			if action.CompletedAt != nil {
				return domain.Conflict("action_completed", "纠正措施 %s 已完成", id)
			}
			seen[id] = true
			resolved = append(resolved, action)
			evidence = append(evidence, proof)
		}
		now := s.now().UTC()
		for i, action := range resolved {
			action.CompletionEvidence = evidence[i]
			action.CompletedAt = &now
		}
		c.AddAudit("correction.batch_completed", cmd.Actor, fmt.Sprintf("批量完成 %d 项纠正措施", len(resolved)), now)
		return nil
	})
}

type AddCorrectiveActionCommand struct {
	WriteMeta
	ID              string    `json:"id"`
	InvestigationID string    `json:"investigationId"`
	Description     string    `json:"description"`
	Owner           string    `json:"owner"`
	DueAt           time.Time `json:"dueAt"`
}

func (s *Service) AddCorrectiveAction(ctx context.Context, campaignID string, cmd AddCorrectiveActionCommand) (*domain.MonitoringCampaign, error) {
	meta, err := mutation(campaignID, "correction.created", cmd.WriteMeta, cmd)
	if err != nil {
		return nil, err
	}
	return s.repo.Update(ctx, meta, func(c *domain.MonitoringCampaign) error {
		if err := c.EnsureMutable(); err != nil {
			return err
		}
		if err := c.RequireStatus(domain.StatusCorrection); err != nil {
			return err
		}
		inv, err := c.Investigation(cmd.InvestigationID)
		if err != nil {
			return err
		}
		if inv.Status != domain.InvestigationConcluded {
			return domain.InvalidState("调查尚未确认根因")
		}
		if strings.TrimSpace(cmd.ID) == "" || strings.TrimSpace(cmd.Description) == "" || strings.TrimSpace(cmd.Owner) == "" || cmd.DueAt.IsZero() {
			return domain.Validation("incomplete_action", "纠正措施 id、description、owner 和 dueAt 均不能为空")
		}
		for _, a := range c.Actions {
			if a.ID == cmd.ID {
				return domain.Conflict("action_exists", "纠正措施 %s 已存在", cmd.ID)
			}
		}
		c.Actions = append(c.Actions, domain.CorrectiveAction{ID: cmd.ID, InvestigationID: cmd.InvestigationID, Description: strings.TrimSpace(cmd.Description), Owner: strings.TrimSpace(cmd.Owner), DueAt: cmd.DueAt.UTC()})
		c.AddAudit("correction.created", cmd.Actor, "已登记纠正措施 "+cmd.ID, s.now().UTC())
		return nil
	})
}

type CompleteCorrectiveActionCommand struct {
	WriteMeta
	CompletionEvidence string `json:"completionEvidence"`
}

func (s *Service) CompleteCorrectiveAction(ctx context.Context, campaignID, actionID string, cmd CompleteCorrectiveActionCommand) (*domain.MonitoringCampaign, error) {
	meta, err := mutation(campaignID, "correction.completed", cmd.WriteMeta, cmd)
	if err != nil {
		return nil, err
	}
	return s.repo.Update(ctx, meta, func(c *domain.MonitoringCampaign) error {
		if err := c.EnsureMutable(); err != nil {
			return err
		}
		if err := c.RequireStatus(domain.StatusCorrection); err != nil {
			return err
		}
		action, err := c.Action(actionID)
		if err != nil {
			return err
		}
		if action.CompletedAt != nil {
			return domain.Conflict("action_completed", "纠正措施 %s 已完成", actionID)
		}
		if strings.TrimSpace(cmd.CompletionEvidence) == "" {
			return domain.Validation("missing_completion_evidence", "completionEvidence 不能为空")
		}
		now := s.now().UTC()
		action.CompletionEvidence, action.CompletedAt = strings.TrimSpace(cmd.CompletionEvidence), &now
		c.AddAudit("correction.completed", cmd.Actor, "纠正措施 "+actionID+" 已完成", now)
		return nil
	})
}

type BeginVerificationCommand struct{ WriteMeta }

func (s *Service) BeginVerification(ctx context.Context, campaignID string, cmd BeginVerificationCommand) (*domain.MonitoringCampaign, error) {
	meta, err := mutation(campaignID, "verification.started", cmd.WriteMeta, cmd)
	if err != nil {
		return nil, err
	}
	return s.repo.Update(ctx, meta, func(c *domain.MonitoringCampaign) error {
		if err := c.EnsureMutable(); err != nil {
			return err
		}
		if err := c.RequireStatus(domain.StatusCorrection); err != nil {
			return err
		}
		complete := assessment.CorrectionCompleteness(c)
		if !complete.Complete {
			return domain.Validation("correction_incomplete", "不能开始验证: %s", strings.Join(complete.Missing, "; "))
		}
		c.VerificationRound++
		if err := c.Transition(domain.StatusVerification); err != nil {
			return err
		}
		c.AddAudit("verification.started", cmd.Actor, "开始验证复采轮次", s.now().UTC())
		return nil
	})
}

package domain

import (
	"fmt"
	"time"
)

func (c *MonitoringCampaign) EnsureMutable() error {
	if c.Status == StatusFrozen || c.FrozenAt != nil {
		return Frozen()
	}
	return nil
}

func (c *MonitoringCampaign) RequireStatus(allowed ...CampaignStatus) error {
	for _, status := range allowed {
		if c.Status == status {
			return nil
		}
	}
	return InvalidState("当前状态 %s 不允许该操作，允许状态为 %v", c.Status, allowed)
}

func (c *MonitoringCampaign) Transition(next CampaignStatus) error {
	if err := c.EnsureMutable(); err != nil {
		return err
	}
	allowed := map[CampaignStatus]map[CampaignStatus]bool{
		StatusDraft:              {StatusPlanLocked: true},
		StatusPlanLocked:         {StatusSampling: true, StatusInvestigation: true},
		StatusSampling:           {StatusSampling: true, StatusInvestigation: true},
		StatusInvestigation:      {StatusInvestigation: true, StatusCorrection: true},
		StatusCorrection:         {StatusCorrection: true, StatusVerification: true},
		StatusVerification:       {StatusVerification: true, StatusInvestigation: true, StatusVerificationPassed: true},
		StatusVerificationPassed: {StatusVerificationPassed: true, StatusFrozen: true},
	}
	if !allowed[c.Status][next] {
		return InvalidState("不能从 %s 转换到 %s", c.Status, next)
	}
	c.Status = next
	return nil
}

func (c *MonitoringCampaign) AddAudit(action, actor, detail string, at time.Time) {
	c.AuditTrail = append(c.AuditTrail, AuditEntry{
		Sequence: int64(len(c.AuditTrail) + 1), Action: action, Actor: actor, Detail: detail, CreatedAt: at,
	})
}

func (c *MonitoringCampaign) HasOpenInvestigation() bool {
	for _, inv := range c.Investigations {
		if inv.Status != InvestigationConcluded {
			return true
		}
	}
	return false
}

func (c *MonitoringCampaign) HasInvestigationForObservation(observationID string) bool {
	for _, inv := range c.Investigations {
		if inv.ObservationID == observationID {
			return true
		}
	}
	return false
}

func (c *MonitoringCampaign) ActionsForInvestigation(investigationID string) []CorrectiveAction {
	var result []CorrectiveAction
	for _, action := range c.Actions {
		if action.InvestigationID == investigationID {
			result = append(result, action)
		}
	}
	return result
}

func (c *MonitoringCampaign) VerificationObservations(round int) []SampleObservation {
	var result []SampleObservation
	for _, observation := range c.Observations {
		if observation.RoundKind == RoundVerification && observation.RoundNumber == round {
			result = append(result, observation)
		}
	}
	return result
}

func (c *MonitoringCampaign) Explain() string {
	return fmt.Sprintf("周期 %s 当前状态 %s，版本 %d", c.ID, c.Status, c.Version)
}

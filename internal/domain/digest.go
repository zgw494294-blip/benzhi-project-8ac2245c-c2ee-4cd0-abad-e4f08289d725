package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
)

type planCanonical struct {
	CampaignID    string           `json:"campaignId"`
	FacilityName  string           `json:"facilityName"`
	Reviewer      string           `json:"reviewer"`
	PlannedRounds int              `json:"plannedRounds"`
	Sites         []ControlledSite `json:"sites"`
}

func (c *MonitoringCampaign) ComputePlanDigest() (string, error) {
	value := planCanonical{CampaignID: c.ID, FacilityName: c.FacilityName, Reviewer: c.PlanReviewer, PlannedRounds: c.PlannedRounds, Sites: c.Sites}
	return digestJSON(value)
}

type frozenCanonical struct {
	ID                string                   `json:"id"`
	FacilityName      string                   `json:"facilityName"`
	Version           int64                    `json:"version"`
	PlanDigest        string                   `json:"planDigest"`
	FrozenAt          any                      `json:"frozenAt"`
	VerificationRound int                      `json:"verificationRound"`
	Sites             []ControlledSite         `json:"sites"`
	Observations      []SampleObservation      `json:"observations"`
	Investigations    []DeviationInvestigation `json:"investigations"`
	Actions           []CorrectiveAction       `json:"actions"`
	Reviews           []QualityReview          `json:"reviews"`
	AuditTrail        []AuditEntry             `json:"auditTrail"`
}

func (c *MonitoringCampaign) ComputeFrozenDigest() (string, error) {
	value := frozenCanonical{
		ID: c.ID, FacilityName: c.FacilityName, Version: c.Version, PlanDigest: c.PlanDigest,
		FrozenAt: c.FrozenAt, VerificationRound: c.VerificationRound, Sites: c.Sites,
		Observations: c.Observations, Investigations: c.Investigations, Actions: c.Actions,
		Reviews: c.Reviews, AuditTrail: c.AuditTrail,
	}
	return digestJSON(value)
}

func digestJSON(value any) (string, error) {
	b, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:]), nil
}

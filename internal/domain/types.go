package domain

import "time"

type CampaignStatus string

const (
	StatusDraft              CampaignStatus = "draft"
	StatusPlanLocked         CampaignStatus = "plan_locked"
	StatusSampling           CampaignStatus = "sampling"
	StatusInvestigation      CampaignStatus = "investigation"
	StatusCorrection         CampaignStatus = "correction"
	StatusVerification       CampaignStatus = "verification"
	StatusVerificationPassed CampaignStatus = "verification_passed"
	StatusFrozen             CampaignStatus = "frozen"
)

type RoundKind string

const (
	RoundPlanned      RoundKind = "planned"
	RoundVerification RoundKind = "verification"
)

type Verdict string

const (
	VerdictNormal Verdict = "normal"
	VerdictAlert  Verdict = "alert"
)

type InvestigationStatus string

const (
	InvestigationOpen      InvestigationStatus = "open"
	InvestigationConcluded InvestigationStatus = "concluded"
)

type MonitoringCampaign struct {
	ID                string                   `json:"id"`
	FacilityName      string                   `json:"facilityName"`
	Status            CampaignStatus           `json:"status"`
	Version           int64                    `json:"version"`
	PlanReviewer      string                   `json:"planReviewer,omitempty"`
	PlannedRounds     int                      `json:"plannedRounds,omitempty"`
	PlanDigest        string                   `json:"planDigest,omitempty"`
	CreatedAt         time.Time                `json:"createdAt"`
	FrozenAt          *time.Time               `json:"frozenAt,omitempty"`
	FrozenDigest      string                   `json:"frozenDigest,omitempty"`
	VerificationRound int                      `json:"verificationRound,omitempty"`
	Sites             []ControlledSite         `json:"sites"`
	Observations      []SampleObservation      `json:"observations"`
	Investigations    []DeviationInvestigation `json:"investigations"`
	Actions           []CorrectiveAction       `json:"actions"`
	Reviews           []QualityReview          `json:"reviews"`
	AuditTrail        []AuditEntry             `json:"auditTrail"`
}

type ControlledSite struct {
	ID               string  `json:"id"`
	CampaignID       string  `json:"campaignId"`
	AreaName         string  `json:"areaName"`
	PointCode        string  `json:"pointCode"`
	CleanlinessGrade string  `json:"cleanlinessGrade"`
	Metric           string  `json:"metric"`
	Unit             string  `json:"unit"`
	AlertLimit       float64 `json:"alertLimit"`
}

type SampleObservation struct {
	ID            string    `json:"id"`
	CampaignID    string    `json:"campaignId"`
	SiteID        string    `json:"siteId"`
	RoundNumber   int       `json:"roundNumber"`
	RoundKind     RoundKind `json:"roundKind"`
	ObservedValue float64   `json:"observedValue"`
	Unit          string    `json:"unit"`
	Verdict       Verdict   `json:"verdict"`
	Explanation   string    `json:"explanation"`
	ObservedAt    time.Time `json:"observedAt"`
}

type DeviationInvestigation struct {
	ID            string              `json:"id"`
	CampaignID    string              `json:"campaignId"`
	ObservationID string              `json:"observationId"`
	ImpactScope   string              `json:"impactScope"`
	Hypotheses    []string            `json:"hypotheses"`
	EvidenceRefs  []string            `json:"evidenceRefs"`
	RootCause     string              `json:"rootCause,omitempty"`
	Status        InvestigationStatus `json:"status"`
	OpenedAt      time.Time           `json:"openedAt"`
	ConcludedAt   *time.Time          `json:"concludedAt,omitempty"`
}

type CorrectiveAction struct {
	ID                 string     `json:"id"`
	InvestigationID    string     `json:"investigationId"`
	Description        string     `json:"description"`
	Owner              string     `json:"owner"`
	DueAt              time.Time  `json:"dueAt"`
	CompletionEvidence string     `json:"completionEvidence,omitempty"`
	CompletedAt        *time.Time `json:"completedAt,omitempty"`
}

type QualityReview struct {
	ID        string    `json:"id"`
	Reviewer  string    `json:"reviewer"`
	Decision  string    `json:"decision"`
	Comment   string    `json:"comment"`
	CreatedAt time.Time `json:"createdAt"`
}

type AuditEntry struct {
	Sequence  int64     `json:"sequence"`
	Action    string    `json:"action"`
	Actor     string    `json:"actor"`
	Detail    string    `json:"detail"`
	CreatedAt time.Time `json:"createdAt"`
}

type ReleaseCredential struct {
	ID              string    `json:"id"`
	CampaignID      string    `json:"campaignId"`
	CampaignVersion int64     `json:"campaignVersion"`
	SnapshotDigest  string    `json:"snapshotDigest"`
	IssuedBy        string    `json:"issuedBy"`
	IssuedAt        time.Time `json:"issuedAt"`
	Signature       string    `json:"signature"`
}

type CredentialVerification struct {
	CredentialID string `json:"credentialId"`
	CampaignID   string `json:"campaignId"`
	Valid        bool   `json:"valid"`
	Reason       string `json:"reason"`
}

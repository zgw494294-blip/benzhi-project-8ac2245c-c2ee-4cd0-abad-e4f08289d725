package domain

import "time"

type ObservationChange struct {
	ObservationID string               `json:"observationId"`
	Before        *PipelineObservation `json:"before,omitempty"`
	After         *PipelineObservation `json:"after,omitempty"`
}

type ControlChange struct {
	ID        string       `json:"id"`
	ControlID string       `json:"controlId"`
	Before    ControlPoint `json:"before"`
	After     ControlPoint `json:"after"`
	Reason    string       `json:"reason"`
	Actor     string       `json:"actor"`
	ChangedAt time.Time    `json:"changedAt"`
}

type Rectification struct {
	ID          string             `json:"id"`
	IssueID     string             `json:"issueId"`
	Note        string             `json:"note"`
	Actor       string             `json:"actor"`
	Change      *ObservationChange `json:"change,omitempty"`
	SubmittedAt time.Time          `json:"submittedAt"`
}

type ReviewDecision struct {
	ID           string    `json:"id"`
	Reviewer     string    `json:"reviewer"`
	Decision     string    `json:"decision"`
	Reason       string    `json:"reason,omitempty"`
	BoundScanID  string    `json:"boundScanId"`
	BoundVersion int64     `json:"boundVersion"`
	DecidedAt    time.Time `json:"decidedAt"`
}

type FrozenSnapshot struct {
	CampaignID     string    `json:"campaignId"`
	FrozenVersion  int64     `json:"frozenVersion"`
	SnapshotDigest string    `json:"snapshotDigest"`
	EventChainRoot string    `json:"eventChainRoot"`
	FrozenAt       time.Time `json:"frozenAt"`
}

type ReleaseCredential struct {
	ID               string    `json:"id"`
	CampaignID       string    `json:"campaignId"`
	FrozenVersion    int64     `json:"frozenVersion"`
	SnapshotDigest   string    `json:"snapshotDigest"`
	EventChainRoot   string    `json:"eventChainRoot"`
	IssuedBy         string    `json:"issuedBy"`
	IssuedAt         time.Time `json:"issuedAt"`
	VerificationCode string    `json:"verificationCode"`
}

type CredentialVerification struct {
	CredentialID  string `json:"credentialId"`
	CampaignID    string `json:"campaignId,omitempty"`
	FrozenVersion int64  `json:"frozenVersion,omitempty"`
	Valid         bool   `json:"valid"`
	Reason        string `json:"reason"`
}

package domain

import "time"

type CampaignState string

const (
	StateDraft          CampaignState = "draft"
	StateBaselineLocked CampaignState = "baseline_locked"
	StateQualityBlocked CampaignState = "quality_blocked"
	StateReadyForReview CampaignState = "ready_for_review"
	StateUnderReview    CampaignState = "under_review"
	StateReturned       CampaignState = "returned"
	StateApproved       CampaignState = "approved"
	StateFrozen         CampaignState = "frozen"
	StateIssued         CampaignState = "issued"
)

type SurveyCampaign struct {
	ID                    string                `json:"id"`
	Name                  string                `json:"name"`
	SurveyArea            string                `json:"surveyArea"`
	CoordinateReference   string                `json:"coordinateReference"`
	SpecificationRevision string                `json:"specificationRevision"`
	State                 CampaignState         `json:"state"`
	Version               int64                 `json:"version"`
	CreatedAt             time.Time             `json:"createdAt"`
	UpdatedAt             time.Time             `json:"updatedAt"`
	Controls              []ControlPoint        `json:"controls"`
	ControlChanges        []ControlChange       `json:"controlChanges"`
	Observations          []PipelineObservation `json:"observations"`
	Issues                []QualityIssue        `json:"issues"`
	Scans                 []QualityScan         `json:"scans"`
	Rectifications        []Rectification       `json:"rectifications"`
	Reviews               []ReviewDecision      `json:"reviews"`
	Frozen                *FrozenSnapshot       `json:"frozen,omitempty"`
	Credential            *ReleaseCredential    `json:"credential,omitempty"`
}

type ControlPoint struct {
	ID         string    `json:"id"`
	CampaignID string    `json:"campaignId"`
	Code       string    `json:"code"`
	Easting    float64   `json:"easting"`
	Northing   float64   `json:"northing"`
	Elevation  float64   `json:"elevation"`
	Source     string    `json:"source"`
	VerifiedBy string    `json:"verifiedBy"`
	VerifiedAt time.Time `json:"verifiedAt"`
}

type PipelineObservation struct {
	ID              string    `json:"id"`
	CampaignID      string    `json:"campaignId"`
	SegmentCode     string    `json:"segmentCode"`
	UtilityType     string    `json:"utilityType"`
	StartPointID    string    `json:"startPointId"`
	EndPointID      string    `json:"endPointId"`
	BurialDepthMM   int       `json:"burialDepthMm"`
	DiameterMM      int       `json:"diameterMm"`
	Material        string    `json:"material"`
	DetectionMethod string    `json:"detectionMethod"`
	ObservedAt      time.Time `json:"observedAt"`
	Revision        int       `json:"revision"`
}

type IssueSeverity string
type IssueStatus string

const (
	SeverityWarning IssueSeverity = "warning"
	SeverityBlocker IssueSeverity = "blocker"
	IssueOpen       IssueStatus   = "open"
	IssueResolved   IssueStatus   = "resolved"
)

type QualityIssue struct {
	ID             string        `json:"id"`
	CampaignID     string        `json:"campaignId"`
	ScanID         string        `json:"scanId"`
	RuleCode       string        `json:"ruleCode"`
	Severity       IssueSeverity `json:"severity"`
	ObjectRef      string        `json:"objectRef"`
	Description    string        `json:"description"`
	Status         IssueStatus   `json:"status"`
	ResolutionNote string        `json:"resolutionNote,omitempty"`
	DetectedAt     time.Time     `json:"detectedAt"`
	ResolvedAt     *time.Time    `json:"resolvedAt,omitempty"`
}

type QualityScan struct {
	ID             string        `json:"id"`
	CampaignID     string        `json:"campaignId"`
	RuleSetVersion string        `json:"ruleSetVersion"`
	InputDigest    string        `json:"inputDigest"`
	IssueCount     int           `json:"issueCount"`
	BlockerCount   int           `json:"blockerCount"`
	ScannedAt      time.Time     `json:"scannedAt"`
	Findings       []ScanFinding `json:"findings"`
}

// ScanFinding 是扫描发生时写入聚合的不可变规则事实，历史扫描不会被复扫覆盖。
type ScanFinding struct {
	Key         string        `json:"key"`
	RuleCode    string        `json:"ruleCode"`
	Severity    IssueSeverity `json:"severity"`
	ObjectRef   string        `json:"objectRef"`
	Description string        `json:"description"`
}

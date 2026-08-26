package domain

import "time"

const (
	DefaultLedgerPageSize  = 20
	MaxLedgerPageSize      = 100
	MaxCorrectionBatchSize = 100
)

func ParseCampaignStatus(value string) (CampaignStatus, error) {
	status := CampaignStatus(value)
	switch status {
	case StatusDraft, StatusPlanLocked, StatusSampling, StatusInvestigation, StatusCorrection,
		StatusVerification, StatusVerificationPassed, StatusFrozen:
		return status, nil
	default:
		return "", Validation("invalid_campaign_status", "未知周期状态 %s", value)
	}
}

func (q *CampaignLedgerQuery) Validate() error {
	if q.Status != nil {
		if _, err := ParseCampaignStatus(string(*q.Status)); err != nil {
			return err
		}
	}
	if q.CreatedFrom != nil && q.CreatedTo != nil && q.CreatedTo.Before(*q.CreatedFrom) {
		return Validation("invalid_created_range", "createdTo 不能早于 createdFrom")
	}
	if q.PageSize == 0 {
		q.PageSize = DefaultLedgerPageSize
	}
	if q.PageSize < 1 || q.PageSize > MaxLedgerPageSize {
		return Validation("invalid_page_size", "pageSize 必须在 1 到 %d 之间", MaxLedgerPageSize)
	}
	return nil
}

type CampaignLedgerSummary struct {
	ID             string         `json:"id"`
	FacilityName   string         `json:"facilityName"`
	Version        int64          `json:"version"`
	Status         CampaignStatus `json:"status"`
	CreatedAt      time.Time      `json:"createdAt"`
	SiteCount      int            `json:"siteCount"`
	AlertCount     int            `json:"alertCount"`
	LastActivityAt time.Time      `json:"lastActivityAt"`
}

type CampaignLedgerPage struct {
	Items      []CampaignLedgerSummary `json:"items"`
	NextCursor string                  `json:"nextCursor,omitempty"`
}

type CampaignStatusStatistics struct {
	ByStatus             map[CampaignStatus]int `json:"byStatus"`
	PendingInvestigation int                    `json:"pendingInvestigation"`
	PendingCorrection    int                    `json:"pendingCorrection"`
	PendingVerification  int                    `json:"pendingVerification"`
	PendingReview        int                    `json:"pendingReview"`
	Total                int                    `json:"total"`
}

type CoverageCellStatus string

const (
	CoverageMissing CoverageCellStatus = "missing"
	CoverageNormal  CoverageCellStatus = "normal"
	CoverageAlert   CoverageCellStatus = "alert"
)

type SamplingCoverageCell struct {
	RoundNumber   int                `json:"roundNumber"`
	SiteID        string             `json:"siteId"`
	AreaName      string             `json:"areaName"`
	PointCode     string             `json:"pointCode"`
	Metric        string             `json:"metric"`
	Status        CoverageCellStatus `json:"status"`
	ObservationID string             `json:"observationId,omitempty"`
	Explanation   string             `json:"explanation,omitempty"`
}

type SamplingRoundSummary struct {
	RoundNumber int `json:"roundNumber"`
	Required    int `json:"required"`
	Recorded    int `json:"recorded"`
	Missing     int `json:"missing"`
	Normal      int `json:"normal"`
	Alert       int `json:"alert"`
}

type AlertWorkItem struct {
	ObservationID       string              `json:"observationId"`
	SiteID              string              `json:"siteId"`
	AreaName            string              `json:"areaName"`
	PointCode           string              `json:"pointCode"`
	Metric              string              `json:"metric"`
	RoundNumber         int                 `json:"roundNumber"`
	InvestigationID     string              `json:"investigationId"`
	InvestigationStatus InvestigationStatus `json:"investigationStatus"`
}

type SamplingProgress struct {
	CampaignID      string                 `json:"campaignId"`
	Version         int64                  `json:"version"`
	Evaluable       bool                   `json:"evaluable"`
	Reason          string                 `json:"reason,omitempty"`
	Cells           []SamplingCoverageCell `json:"cells"`
	Rounds          []SamplingRoundSummary `json:"rounds"`
	CompletionRatio float64                `json:"completionRatio"`
	AlertWorkItems  []AlertWorkItem        `json:"alertWorkItems"`
}

type SamplingProgressFilter struct {
	RoundNumber int
	AreaName    string
	Metric      string
}

type InvestigationDraftPatch struct {
	ImpactScope  *string
	Hypotheses   *[]string
	EvidenceRefs *[]string
}

type InvestigationClosePreflight struct {
	CampaignID     string                 `json:"campaignId"`
	Version        int64                  `json:"version"`
	CampaignStatus CampaignStatus         `json:"campaignStatus"`
	Investigation  DeviationInvestigation `json:"investigation"`
	Observation    SampleObservation      `json:"observation"`
	MissingFields  []string               `json:"missingFields"`
	CanConclude    bool                   `json:"canConclude"`
}

type CorrectiveActionFilter struct {
	InvestigationID string
	Owner           string
	Completed       *bool
	Overdue         *bool
}

type CorrectiveActionListItem struct {
	CorrectiveAction
	Completed bool `json:"completed"`
	Overdue   bool `json:"overdue"`
}

type CorrectiveActionStatistics struct {
	Total       int `json:"total"`
	Completed   int `json:"completed"`
	Outstanding int `json:"outstanding"`
	Overdue     int `json:"overdue"`
}

type CorrectiveActionList struct {
	CampaignID string                     `json:"campaignId"`
	Version    int64                      `json:"version"`
	AsOf       time.Time                  `json:"asOf"`
	Items      []CorrectiveActionListItem `json:"items"`
	Summary    CorrectiveActionStatistics `json:"summary"`
}

type VerificationBlocker struct {
	Code            string `json:"code"`
	InvestigationID string `json:"investigationId,omitempty"`
	ActionID        string `json:"actionId,omitempty"`
	Message         string `json:"message"`
}

type VerificationStartPreflight struct {
	CampaignID string                `json:"campaignId"`
	Version    int64                 `json:"version"`
	Status     CampaignStatus        `json:"status"`
	CanStart   bool                  `json:"canStart"`
	Blockers   []VerificationBlocker `json:"blockers"`
}

type VerificationCell struct {
	RoundNumber     int     `json:"roundNumber"`
	Missing         bool    `json:"missing"`
	ObservationID   string  `json:"observationId,omitempty"`
	ObservedValue   float64 `json:"observedValue,omitempty"`
	Unit            string  `json:"unit,omitempty"`
	AlertLimit      float64 `json:"alertLimit"`
	Verdict         Verdict `json:"verdict,omitempty"`
	InvestigationID string  `json:"investigationId,omitempty"`
}

type VerificationSiteComparison struct {
	SiteID    string             `json:"siteId"`
	AreaName  string             `json:"areaName"`
	PointCode string             `json:"pointCode"`
	Metric    string             `json:"metric"`
	Rounds    []VerificationCell `json:"rounds"`
}

type VerificationRoundSummary struct {
	RoundNumber      int      `json:"roundNumber"`
	CoverageRatio    float64  `json:"coverageRatio"`
	Normal           int      `json:"normal"`
	Alert            int      `json:"alert"`
	Missing          int      `json:"missing"`
	Conclusion       string   `json:"conclusion"`
	InvestigationIDs []string `json:"investigationIds,omitempty"`
}

type VerificationComparison struct {
	CampaignID string                       `json:"campaignId"`
	Version    int64                        `json:"version"`
	Status     CampaignStatus               `json:"status"`
	Reason     string                       `json:"reason,omitempty"`
	Sites      []VerificationSiteComparison `json:"sites"`
	Rounds     []VerificationRoundSummary   `json:"rounds"`
}

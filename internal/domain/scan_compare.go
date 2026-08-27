package domain

import "time"

type DifferenceCounts struct {
	Resolved   int `json:"resolved"`
	Persistent int `json:"persistent"`
	New        int `json:"new"`
}

type ScanDifferenceItem struct {
	Finding        ScanFinding    `json:"finding"`
	ResolutionType string         `json:"resolutionType,omitempty"`
	Rectification  *Rectification `json:"rectification,omitempty"`
}

type ScanDifferenceReport struct {
	CampaignID   string                      `json:"campaignId"`
	Version      int64                       `json:"version"`
	BaseScanID   string                      `json:"baseScanId"`
	TargetScanID string                      `json:"targetScanId"`
	Summary      DifferenceCounts            `json:"summary"`
	ByRule       map[string]DifferenceCounts `json:"byRule"`
	Resolved     []ScanDifferenceItem        `json:"resolved"`
	Persistent   []ScanDifferenceItem        `json:"persistent"`
	New          []ScanDifferenceItem        `json:"new"`
}

func FindingKey(ruleCode, objectRef string) string {
	return ruleCode + "|" + objectRef
}

func IssueIDForFinding(ruleCode, objectRef string) string {
	return "iss_" + Digest(FindingKey(ruleCode, objectRef))[:24]
}

func (c *SurveyCampaign) ScanIndex(id string) int {
	for i := range c.Scans {
		if c.Scans[i].ID == id {
			return i
		}
	}
	return -1
}

func (c *SurveyCampaign) LatestRectificationBefore(issueID string, scanAt time.Time) *Rectification {
	var latest *Rectification
	for i := range c.Rectifications {
		r := &c.Rectifications[i]
		if r.IssueID != issueID || r.SubmittedAt.After(scanAt) {
			continue
		}
		if latest == nil || r.SubmittedAt.After(latest.SubmittedAt) || (r.SubmittedAt.Equal(latest.SubmittedAt) && r.ID > latest.ID) {
			copy := *r
			latest = &copy
		}
	}
	return latest
}

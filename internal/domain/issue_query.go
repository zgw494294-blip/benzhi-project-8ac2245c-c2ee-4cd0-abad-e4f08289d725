package domain

import (
	"sort"
	"strings"
	"time"
)

type IssueFilter struct {
	Status    IssueStatus   `json:"status,omitempty"`
	Severity  IssueSeverity `json:"severity,omitempty"`
	RuleCode  string        `json:"ruleCode,omitempty"`
	ObjectRef string        `json:"objectRef,omitempty"`
}

type IssueWorkItem struct {
	ID                      string        `json:"id"`
	RuleCode                string        `json:"ruleCode"`
	Severity                IssueSeverity `json:"severity"`
	ObjectRef               string        `json:"objectRef"`
	Description             string        `json:"description"`
	Status                  IssueStatus   `json:"status"`
	LatestScanID            string        `json:"latestScanId"`
	DetectedAt              time.Time     `json:"detectedAt"`
	ResolvedAt              *time.Time    `json:"resolvedAt"`
	LatestRectificationNote string        `json:"latestRectificationNote"`
}

type IssueSummary struct {
	OpenCount     int            `json:"openCount"`
	ResolvedCount int            `json:"resolvedCount"`
	BlockerCount  int            `json:"blockerCount"`
	WarningCount  int            `json:"warningCount"`
	ByRuleCode    map[string]int `json:"byRuleCode"`
}

type IssueWorklist struct {
	CampaignID   string          `json:"campaignId"`
	Version      int64           `json:"version"`
	LatestScanID string          `json:"latestScanId"`
	Items        []IssueWorkItem `json:"items"`
	Summary      IssueSummary    `json:"summary"`
	NextCursor   string          `json:"nextCursor"`
}

func (c *SurveyCampaign) ValidateIssueFilter(filter IssueFilter) error {
	if filter.Status != "" && filter.Status != IssueOpen && filter.Status != IssueResolved {
		return QueryError("status", "未知的问题状态")
	}
	if filter.Severity != "" && filter.Severity != SeverityBlocker && filter.Severity != SeverityWarning {
		return QueryError("severity", "未知的问题严重度")
	}
	if filter.ObjectRef != "" {
		found := false
		for _, issue := range c.Issues {
			found = found || issue.ObjectRef == filter.ObjectRef
		}
		if !found && !c.ownsObjectRef(filter.ObjectRef) {
			return QueryError("objectRef", "问题引用不属于当前批次")
		}
	}
	return nil
}

func (c *SurveyCampaign) ownsObjectRef(ref string) bool {
	kind, value, ok := strings.Cut(ref, ":")
	if !ok || value == "" {
		return false
	}
	switch kind {
	case "campaign":
		return value == c.ID
	case "control":
		return c.HasControl(value)
	case "observation":
		return c.ObservationIndex(value) >= 0
	case "controls", "observations":
		ids := strings.Split(value, ",")
		if len(ids) < 2 {
			return false
		}
		for _, id := range ids {
			if kind == "controls" && !c.HasControl(id) || kind == "observations" && c.ObservationIndex(id) < 0 {
				return false
			}
		}
		return true
	default:
		return false
	}
}

func (c *SurveyCampaign) IssueItems(filter IssueFilter) []IssueWorkItem {
	items := make([]IssueWorkItem, 0, len(c.Issues))
	for _, issue := range c.Issues {
		if filter.Status != "" && issue.Status != filter.Status || filter.Severity != "" && issue.Severity != filter.Severity || filter.RuleCode != "" && issue.RuleCode != filter.RuleCode || filter.ObjectRef != "" && issue.ObjectRef != filter.ObjectRef {
			continue
		}
		item := IssueWorkItem{ID: issue.ID, RuleCode: issue.RuleCode, Severity: issue.Severity, ObjectRef: issue.ObjectRef, Description: issue.Description, Status: issue.Status, LatestScanID: issue.ScanID, DetectedAt: issue.DetectedAt, ResolvedAt: issue.ResolvedAt}
		for i := len(c.Rectifications) - 1; i >= 0; i-- {
			if c.Rectifications[i].IssueID == issue.ID {
				item.LatestRectificationNote = c.Rectifications[i].Note
				break
			}
		}
		items = append(items, item)
	}
	sort.Slice(items, func(i, j int) bool {
		iPriority := items[i].Status == IssueOpen && items[i].Severity == SeverityBlocker
		jPriority := items[j].Status == IssueOpen && items[j].Severity == SeverityBlocker
		if iPriority != jPriority {
			return iPriority
		}
		if !items[i].DetectedAt.Equal(items[j].DetectedAt) {
			return items[i].DetectedAt.Before(items[j].DetectedAt)
		}
		return items[i].ID < items[j].ID
	})
	return items
}

func (c *SurveyCampaign) IssueSummary() IssueSummary {
	summary := IssueSummary{ByRuleCode: map[string]int{}}
	for _, issue := range c.Issues {
		if issue.Status == IssueOpen {
			summary.OpenCount++
		} else {
			summary.ResolvedCount++
		}
		if issue.Severity == SeverityBlocker {
			summary.BlockerCount++
		} else {
			summary.WarningCount++
		}
		summary.ByRuleCode[issue.RuleCode]++
	}
	return summary
}

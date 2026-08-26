package quality

import (
	"fmt"
	"math"
	"sort"
	"time"

	"subsurface-survey-gate/internal/domain"
)

func (s *Scanner) BaselineReadiness(c *domain.SurveyCampaign) domain.BaselineReadiness {
	issues := make([]domain.ReadinessIssue, 0)
	if len(c.Controls) < 2 {
		issues = append(issues, domain.ReadinessIssue{RuleCode: "CTRL_MINIMUM", Severity: domain.SeverityBlocker, ObjectRef: "campaign:" + c.ID, ControlCodes: []string{}, Description: "控制点少于两个"})
	}
	controls := append([]domain.ControlPoint(nil), c.Controls...)
	sort.Slice(controls, func(i, j int) bool {
		if controls[i].Code != controls[j].Code {
			return controls[i].Code < controls[j].Code
		}
		return controls[i].ID < controls[j].ID
	})
	for i, p := range controls {
		if p.VerifiedBy == "" {
			issues = append(issues, domain.ReadinessIssue{RuleCode: "CTRL_UNVERIFIED", Severity: domain.SeverityBlocker, ObjectRef: "control:" + p.ID, ControlCodes: []string{p.Code}, Field: "verifiedBy", Description: "控制点缺少核验人"})
		}
		if p.VerifiedAt.IsZero() || p.VerifiedAt.After(c.UpdatedAt.Add(5*time.Minute)) {
			issues = append(issues, domain.ReadinessIssue{RuleCode: "CTRL_UNVERIFIED", Severity: domain.SeverityBlocker, ObjectRef: "control:" + p.ID, ControlCodes: []string{p.Code}, Field: "verifiedAt", Description: "控制点核验时间无效"})
		}
		for j := i + 1; j < len(controls); j++ {
			q := controls[j]
			if math.Hypot(p.Easting-q.Easting, p.Northing-q.Northing) < 0.001 {
				codes := []string{p.Code, q.Code}
				issues = append(issues, domain.ReadinessIssue{RuleCode: "CTRL_CLOSURE", Severity: domain.SeverityBlocker, ObjectRef: fmt.Sprintf("controls:%s,%s", p.ID, q.ID), ControlCodes: codes, Description: fmt.Sprintf("控制点 %s 与 %s 的平面坐标重合，无法形成闭合基准", codes[0], codes[1])})
			}
		}
	}
	sort.Slice(issues, func(i, j int) bool {
		if issues[i].RuleCode != issues[j].RuleCode {
			return issues[i].RuleCode < issues[j].RuleCode
		}
		if issues[i].ObjectRef != issues[j].ObjectRef {
			return issues[i].ObjectRef < issues[j].ObjectRef
		}
		if issues[i].Field != issues[j].Field {
			return issues[i].Field < issues[j].Field
		}
		return issues[i].Description < issues[j].Description
	})
	return domain.BaselineReadiness{CampaignID: c.ID, Version: c.Version, ReadyToLock: len(issues) == 0, ReadinessDigest: domain.BaselineReadinessDigest(c), Summary: domain.ControlReadinessSummary(c), Issues: issues}
}

package assessment

import (
	"fmt"

	"cleanroom-release-go/internal/domain"
)

type VerificationResult struct {
	Complete          bool     `json:"complete"`
	Passed            bool     `json:"passed"`
	MissingSites      []string `json:"missingSites,omitempty"`
	AlertObservations []string `json:"alertObservations,omitempty"`
	Reason            string   `json:"reason"`
}

func AssessVerification(c *domain.MonitoringCampaign, round int) VerificationResult {
	result := VerificationResult{Passed: true}
	observations := c.VerificationObservations(round)
	bySite := map[string]domain.SampleObservation{}
	for _, observation := range observations {
		bySite[observation.SiteID] = observation
		if observation.Verdict == domain.VerdictAlert {
			result.AlertObservations = append(result.AlertObservations, observation.ID)
		}
	}
	for _, site := range c.Sites {
		if _, ok := bySite[site.ID]; !ok {
			result.MissingSites = append(result.MissingSites, site.ID)
		}
	}
	result.Complete = len(result.MissingSites) == 0
	result.Passed = result.Complete && len(result.AlertObservations) == 0
	switch {
	case len(result.AlertObservations) > 0:
		result.Reason = fmt.Sprintf("验证轮次 %d 存在 %d 个再次超限结果", round, len(result.AlertObservations))
	case !result.Complete:
		result.Reason = fmt.Sprintf("验证轮次 %d 尚缺少 %d 个监测点结果", round, len(result.MissingSites))
	default:
		result.Reason = fmt.Sprintf("验证轮次 %d 已覆盖全部监测点且均合格", round)
	}
	return result
}

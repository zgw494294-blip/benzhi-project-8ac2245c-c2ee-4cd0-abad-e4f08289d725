package assessment

import (
	"cleanroom-release-go/internal/domain"
)

func ReviewReadiness(c *domain.MonitoringCampaign) Completeness {
	result := Completeness{Complete: true}
	if c.PlanDigest == "" {
		result.Missing = append(result.Missing, "采样方案摘要")
	}
	planned := PlannedSamplingCompleteness(c)
	result.Missing = append(result.Missing, planned.Missing...)
	corrections := CorrectionCompleteness(c)
	result.Missing = append(result.Missing, corrections.Missing...)
	verification := AssessVerification(c, c.VerificationRound)
	if !verification.Passed {
		result.Missing = append(result.Missing, verification.Reason)
	}
	result.Complete = len(result.Missing) == 0
	return result
}

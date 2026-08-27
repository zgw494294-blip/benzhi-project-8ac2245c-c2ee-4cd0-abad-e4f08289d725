package quality

import (
	"sort"

	"subsurface-survey-gate/internal/domain"
)

type Scanner struct{}

func NewScanner() *Scanner { return &Scanner{} }

func (s *Scanner) Scan(c *domain.SurveyCampaign) Result {
	findings := make([]Finding, 0)
	findings = append(findings, baselineFindings(c)...)
	findings = append(findings, observationFindings(c)...)
	findings = append(findings, geometryFindings(c)...)
	findings = append(findings, depthFindings(c)...)
	sort.Slice(findings, func(i, j int) bool {
		if findings[i].RuleCode != findings[j].RuleCode {
			return findings[i].RuleCode < findings[j].RuleCode
		}
		if findings[i].ObjectRef != findings[j].ObjectRef {
			return findings[i].ObjectRef < findings[j].ObjectRef
		}
		return findings[i].Description < findings[j].Description
	})
	input := struct {
		Controls     []domain.ControlPoint        `json:"controls"`
		Observations []domain.PipelineObservation `json:"observations"`
	}{c.Controls, c.Observations}
	return Result{RuleSetVersion: RuleSetVersion, InputDigest: domain.Digest(input), Findings: findings}
}

package quality

import "subsurface-survey-gate/internal/domain"

const RuleSetVersion = "survey-quality-rules-1.0"

type Finding struct {
	RuleCode    string
	Severity    domain.IssueSeverity
	ObjectRef   string
	Description string
}

type Result struct {
	RuleSetVersion string
	InputDigest    string
	Findings       []Finding
}

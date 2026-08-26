package assessment

import (
	"strings"

	"cleanroom-release-go/internal/domain"
)

type Completeness struct {
	Complete bool     `json:"complete"`
	Missing  []string `json:"missing"`
}

func InvestigationCompleteness(inv domain.DeviationInvestigation) Completeness {
	result := Completeness{Complete: true}
	if strings.TrimSpace(inv.ImpactScope) == "" {
		result.Missing = append(result.Missing, "impactScope")
	}
	if len(nonBlank(inv.Hypotheses)) == 0 {
		result.Missing = append(result.Missing, "hypotheses")
	}
	if len(nonBlank(inv.EvidenceRefs)) == 0 {
		result.Missing = append(result.Missing, "evidenceRefs")
	}
	if strings.TrimSpace(inv.RootCause) == "" {
		result.Missing = append(result.Missing, "rootCause")
	}
	result.Complete = len(result.Missing) == 0
	return result
}

func nonBlank(values []string) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			result = append(result, value)
		}
	}
	return result
}

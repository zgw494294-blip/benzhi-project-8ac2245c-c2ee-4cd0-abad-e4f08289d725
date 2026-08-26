package assessment

import (
	"fmt"
	"strings"

	"cleanroom-release-go/internal/domain"
)

func CorrectionCompleteness(c *domain.MonitoringCampaign) Completeness {
	result := Completeness{Complete: true}
	for _, inv := range c.Investigations {
		if inv.Status != domain.InvestigationConcluded {
			result.Missing = append(result.Missing, fmt.Sprintf("调查 %s 尚未闭合", inv.ID))
			continue
		}
		actions := c.ActionsForInvestigation(inv.ID)
		if len(actions) == 0 {
			result.Missing = append(result.Missing, fmt.Sprintf("调查 %s 未制定纠正措施", inv.ID))
		}
		for _, action := range actions {
			if strings.TrimSpace(action.Owner) == "" || strings.TrimSpace(action.Description) == "" {
				result.Missing = append(result.Missing, fmt.Sprintf("纠正措施 %s 资料不完整", action.ID))
			}
			if action.CompletedAt == nil || strings.TrimSpace(action.CompletionEvidence) == "" {
				result.Missing = append(result.Missing, fmt.Sprintf("纠正措施 %s 尚未完成", action.ID))
			}
		}
	}
	result.Complete = len(result.Missing) == 0
	return result
}

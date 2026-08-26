package assessment

import (
	"fmt"

	"cleanroom-release-go/internal/domain"
)

// PlannedSamplingCompleteness 核对锁定方案中的每个轮次、点位和指标都已留下唯一结果。
func PlannedSamplingCompleteness(c *domain.MonitoringCampaign) Completeness {
	result := Completeness{Complete: true}
	seen := make(map[string]bool, len(c.Observations))
	for _, observation := range c.Observations {
		if observation.RoundKind != domain.RoundPlanned {
			continue
		}
		key := fmt.Sprintf("%d|%s", observation.RoundNumber, observation.SiteID)
		seen[key] = true
	}
	for round := 1; round <= c.PlannedRounds; round++ {
		for _, site := range c.Sites {
			key := fmt.Sprintf("%d|%s", round, site.ID)
			if !seen[key] {
				result.Missing = append(result.Missing, fmt.Sprintf("计划轮次 %d 缺少监测点 %s 的 %s 结果", round, site.ID, site.Metric))
			}
		}
	}
	result.Complete = len(result.Missing) == 0
	return result
}

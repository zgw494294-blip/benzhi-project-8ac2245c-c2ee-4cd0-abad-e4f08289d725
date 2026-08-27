package quality

import (
	"fmt"
	"math"
	"sort"

	"subsurface-survey-gate/internal/domain"
)

func depthFindings(c *domain.SurveyCampaign) []Finding {
	obs := append([]domain.PipelineObservation(nil), c.Observations...)
	sort.Slice(obs, func(i, j int) bool { return obs[i].ID < obs[j].ID })
	var out []Finding
	for i := 0; i < len(obs); i++ {
		for j := i + 1; j < len(obs); j++ {
			if !connected(obs[i], obs[j]) {
				continue
			}
			delta := int(math.Abs(float64(obs[i].BurialDepthMM - obs[j].BurialDepthMM)))
			if delta > 4000 {
				out = append(out, Finding{"DEPTH_JUMP", domain.SeverityBlocker, fmt.Sprintf("observations:%s,%s", obs[i].ID, obs[j].ID), fmt.Sprintf("相邻管段埋深突变 %d mm", delta)})
			} else if delta > 2000 {
				out = append(out, Finding{"DEPTH_VARIATION", domain.SeverityWarning, fmt.Sprintf("observations:%s,%s", obs[i].ID, obs[j].ID), fmt.Sprintf("相邻管段埋深变化 %d mm，建议复核", delta)})
			}
		}
	}
	return out
}

func connected(a, b domain.PipelineObservation) bool {
	return a.StartPointID == b.StartPointID || a.StartPointID == b.EndPointID || a.EndPointID == b.StartPointID || a.EndPointID == b.EndPointID
}

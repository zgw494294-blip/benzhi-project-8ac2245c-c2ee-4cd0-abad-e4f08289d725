package quality

import (
	"fmt"
	"sort"

	"subsurface-survey-gate/internal/domain"
)

func geometryFindings(c *domain.SurveyCampaign) []Finding {
	seen := map[string]string{}
	var out []Finding
	for _, o := range c.Observations {
		ends := []string{o.StartPointID, o.EndPointID}
		sort.Strings(ends)
		key := ends[0] + "|" + ends[1]
		if prior, ok := seen[key]; ok {
			out = append(out, Finding{"GEOMETRY_DUPLICATE", domain.SeverityBlocker, fmt.Sprintf("observations:%s,%s", prior, o.ID), "两条管段具有相同的控制点几何关系"})
		} else {
			seen[key] = o.ID
		}
	}
	return out
}

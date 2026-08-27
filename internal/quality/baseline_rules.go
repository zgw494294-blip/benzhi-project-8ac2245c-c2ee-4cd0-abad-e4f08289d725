package quality

import (
	"fmt"
	"math"
	"time"

	"subsurface-survey-gate/internal/domain"
)

func baselineFindings(c *domain.SurveyCampaign) []Finding {
	var out []Finding
	if len(c.Controls) < 2 {
		out = append(out, Finding{"CTRL_MINIMUM", domain.SeverityBlocker, "campaign:" + c.ID, "控制点少于两个"})
	}
	for i, p := range c.Controls {
		if p.VerifiedBy == "" || p.VerifiedAt.IsZero() || p.VerifiedAt.After(c.UpdatedAt.Add(5*time.Minute)) {
			out = append(out, Finding{"CTRL_UNVERIFIED", domain.SeverityBlocker, "control:" + p.ID, "控制点缺少核验信息"})
		}
		for j := i + 1; j < len(c.Controls); j++ {
			q := c.Controls[j]
			d := math.Hypot(p.Easting-q.Easting, p.Northing-q.Northing)
			if d < 0.001 {
				out = append(out, Finding{"CTRL_CLOSURE", domain.SeverityBlocker, fmt.Sprintf("controls:%s,%s", p.ID, q.ID), "两个控制点平面坐标重合，无法形成闭合基准"})
			}
		}
	}
	return out
}

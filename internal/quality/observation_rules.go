package quality

import (
	"fmt"

	"subsurface-survey-gate/internal/domain"
)

func observationFindings(c *domain.SurveyCampaign) []Finding {
	var out []Finding
	if len(c.Observations) == 0 {
		out = append(out, Finding{"OBS_REQUIRED", domain.SeverityBlocker, "campaign:" + c.ID, "至少需要一条管段观测"})
	}
	codes := map[string]string{}
	for _, o := range c.Observations {
		ref := "observation:" + o.ID
		if !c.HasControl(o.StartPointID) || !c.HasControl(o.EndPointID) {
			out = append(out, Finding{"ENDPOINT_REFERENCE", domain.SeverityBlocker, ref, "管段引用了不存在的控制点"})
		}
		if o.UtilityType == "" || o.Material == "" || o.DetectionMethod == "" {
			out = append(out, Finding{"REQUIRED_ATTRIBUTES", domain.SeverityBlocker, ref, "管线类别、材质和探测方法均为必填"})
		}
		if o.BurialDepthMM < 0 || o.BurialDepthMM > 30000 || o.DiameterMM <= 0 || o.DiameterMM > 10000 {
			out = append(out, Finding{"VALUE_RANGE", domain.SeverityBlocker, ref, "埋深或管径超出允许范围"})
		}
		if prior, ok := codes[o.SegmentCode]; ok {
			out = append(out, Finding{"SEGMENT_DUPLICATE", domain.SeverityBlocker, fmt.Sprintf("observations:%s,%s", prior, o.ID), "管段段号重复"})
		} else {
			codes[o.SegmentCode] = o.ID
		}
	}
	return out
}

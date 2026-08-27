package domain

import "encoding/json"

func Clone(c *SurveyCampaign) *SurveyCampaign {
	if c == nil {
		return nil
	}
	b, _ := json.Marshal(c)
	var out SurveyCampaign
	_ = json.Unmarshal(b, &out)
	return &out
}

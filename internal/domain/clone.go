package domain

import "encoding/json"

func CloneCampaign(c *MonitoringCampaign) (*MonitoringCampaign, error) {
	b, err := json.Marshal(c)
	if err != nil {
		return nil, err
	}
	var clone MonitoringCampaign
	if err := json.Unmarshal(b, &clone); err != nil {
		return nil, err
	}
	return &clone, nil
}

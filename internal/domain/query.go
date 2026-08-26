package domain

func (c *SurveyCampaign) HasControl(id string) bool {
	for _, p := range c.Controls {
		if p.ID == id {
			return true
		}
	}
	return false
}

func (c *SurveyCampaign) ObservationIndex(id string) int {
	for i := range c.Observations {
		if c.Observations[i].ID == id {
			return i
		}
	}
	return -1
}

func (c *SurveyCampaign) FindIssue(id string) *QualityIssue {
	for i := range c.Issues {
		if c.Issues[i].ID == id {
			return &c.Issues[i]
		}
	}
	return nil
}

func (c *SurveyCampaign) OpenBlockerCount() int {
	n := 0
	for _, issue := range c.Issues {
		if issue.Status == IssueOpen && issue.Severity == SeverityBlocker {
			n++
		}
	}
	return n
}

func (c *SurveyCampaign) LatestScan() *QualityScan {
	if len(c.Scans) == 0 {
		return nil
	}
	return &c.Scans[len(c.Scans)-1]
}

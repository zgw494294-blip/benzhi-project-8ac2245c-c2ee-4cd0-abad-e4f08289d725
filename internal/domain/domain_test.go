package domain

import "testing"

func TestValidateCampaignAndTransitions(t *testing.T) {
	c := &MonitoringCampaign{ID: "c1", FacilityName: "厂房", Status: StatusDraft, Sites: []ControlledSite{{ID: "s1", AreaName: "A区", PointCode: "P1", CleanlinessGrade: "B", Metric: "particle_0_5um", Unit: "particles/m3", AlertLimit: 100}}}
	if err := ValidateNewCampaign(c); err != nil {
		t.Fatal(err)
	}
	if c.Sites[0].CampaignID != "c1" {
		t.Fatal("应自动关联周期 id")
	}
	if err := c.Transition(StatusPlanLocked); err != nil {
		t.Fatal(err)
	}
	if err := c.Transition(StatusCorrection); err == nil {
		t.Fatal("非法跨状态转换应失败")
	}
	if err := c.Transition(StatusSampling); err != nil {
		t.Fatal(err)
	}
}

func TestStableDigests(t *testing.T) {
	c := &MonitoringCampaign{ID: "c1", FacilityName: "厂房", PlanReviewer: "复核员", PlannedRounds: 1, Sites: []ControlledSite{{ID: "s1", Metric: "settle_plate", Unit: "cfu/plate", AlertLimit: 2}}}
	a, err := c.ComputePlanDigest()
	if err != nil {
		t.Fatal(err)
	}
	b, err := c.ComputePlanDigest()
	if err != nil {
		t.Fatal(err)
	}
	if a == "" || a != b {
		t.Fatalf("方案摘要不稳定: %q %q", a, b)
	}
	c.Sites[0].AlertLimit = 3
	c2, err := c.ComputePlanDigest()
	if err != nil {
		t.Fatal(err)
	}
	if c2 == a {
		t.Fatal("限值变化应改变方案摘要")
	}
}

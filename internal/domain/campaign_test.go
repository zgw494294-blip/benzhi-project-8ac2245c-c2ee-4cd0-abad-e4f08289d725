package domain

import (
	"testing"
	"time"
)

func TestCampaignBaselineAndObservationInvariants(t *testing.T) {
	now := time.Now().UTC().Add(-time.Minute)
	c, err := NewCampaign("cmp-1", "道路探测", "A区", "CGCS2000", "r1", now)
	if err != nil {
		t.Fatal(err)
	}
	p1 := ControlPoint{ID: "p1", Code: "P1", Easting: 1, Northing: 1, Elevation: 1, Source: "GNSS", VerifiedBy: "甲", VerifiedAt: now}
	p2 := ControlPoint{ID: "p2", Code: "P2", Easting: 2, Northing: 2, Elevation: 1, Source: "GNSS", VerifiedBy: "甲", VerifiedAt: now}
	if err := c.AddControl(p1, now); err != nil {
		t.Fatal(err)
	}
	if err := c.LockBaseline(now); err == nil {
		t.Fatal("一个控制点不应允许锁定")
	}
	if err := c.AddControl(p2, now); err != nil {
		t.Fatal(err)
	}
	if err := c.LockBaseline(now); err != nil {
		t.Fatal(err)
	}
	o := PipelineObservation{ID: "o1", SegmentCode: "S1", UtilityType: "给水", StartPointID: "p1", EndPointID: "missing", BurialDepthMM: 1000, DiameterMM: 100, Material: "钢", DetectionMethod: "电磁", ObservedAt: now}
	if err := c.AddObservation(o, now); err == nil {
		t.Fatal("悬空端点不应通过")
	}
	o.EndPointID = "p2"
	if err := c.AddObservation(o, now); err != nil {
		t.Fatal(err)
	}
	duplicate := o
	duplicate.ID = "o2"
	if err := c.AddObservation(duplicate, now); err == nil {
		t.Fatal("重复段号不应通过")
	}
}

func TestRectificationDoesNotDirectlyResolveIssue(t *testing.T) {
	now := time.Now().UTC().Add(-time.Minute)
	c := &SurveyCampaign{ID: "c", State: StateQualityBlocked, Version: 3, Issues: []QualityIssue{{ID: "i", Status: IssueOpen, Severity: SeverityBlocker}}, Observations: []PipelineObservation{{ID: "o", SegmentCode: "S", UtilityType: "给水", StartPointID: "p1", EndPointID: "p2", BurialDepthMM: 900, DiameterMM: 100, Material: "钢", DetectionMethod: "电磁", ObservedAt: now}}, Controls: []ControlPoint{{ID: "p1"}, {ID: "p2"}}}
	revised := c.Observations[0]
	revised.BurialDepthMM = 1000
	if err := c.ReviseObservation("i", "fix", "复测修订", "甲", &revised, now); err != nil {
		t.Fatal(err)
	}
	if c.Issues[0].Status != IssueOpen {
		t.Fatal("整改提交不得直接关闭问题")
	}
	if c.Rectifications[0].Change.Before.BurialDepthMM != 900 || c.Rectifications[0].Change.After.BurialDepthMM != 1000 {
		t.Fatal("未保留修订前后值")
	}
}

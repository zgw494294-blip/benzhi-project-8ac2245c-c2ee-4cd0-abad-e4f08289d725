package quality

import (
	"reflect"
	"testing"
	"time"

	"subsurface-survey-gate/internal/domain"
)

func TestScannerDeterministicDepthAndGeometryRules(t *testing.T) {
	now := time.Now().UTC()
	c := &domain.SurveyCampaign{ID: "c", Controls: []domain.ControlPoint{{ID: "p1", VerifiedBy: "甲", VerifiedAt: now}, {ID: "p2", Easting: 10, Northing: 10, VerifiedBy: "甲", VerifiedAt: now}, {ID: "p3", Easting: 20, Northing: 20, VerifiedBy: "甲", VerifiedAt: now}}, Observations: []domain.PipelineObservation{
		{ID: "b", SegmentCode: "B", UtilityType: "给水", StartPointID: "p2", EndPointID: "p3", BurialDepthMM: 6500, DiameterMM: 100, Material: "钢", DetectionMethod: "雷达"},
		{ID: "a", SegmentCode: "A", UtilityType: "给水", StartPointID: "p1", EndPointID: "p2", BurialDepthMM: 1000, DiameterMM: 100, Material: "钢", DetectionMethod: "雷达"},
		{ID: "d", SegmentCode: "D", UtilityType: "给水", StartPointID: "p2", EndPointID: "p1", BurialDepthMM: 1200, DiameterMM: 100, Material: "钢", DetectionMethod: "雷达"},
	}}
	s := NewScanner()
	first, second := s.Scan(c), s.Scan(c)
	if !reflect.DeepEqual(first, second) {
		t.Fatal("同一输入扫描结果不稳定")
	}
	codes := map[string]bool{}
	for _, f := range first.Findings {
		codes[f.RuleCode] = true
	}
	if !codes["DEPTH_JUMP"] || !codes["GEOMETRY_DUPLICATE"] {
		t.Fatalf("缺少预期规则: %#v", codes)
	}
}

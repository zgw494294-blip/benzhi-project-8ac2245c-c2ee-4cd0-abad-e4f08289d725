package domain

import (
	"math"
	"strings"
	"time"
)

func ValidateCampaign(c SurveyCampaign) error {
	if strings.TrimSpace(c.Name) == "" {
		return Validation("name", "不能为空")
	}
	if strings.TrimSpace(c.SurveyArea) == "" {
		return Validation("surveyArea", "不能为空")
	}
	if strings.TrimSpace(c.CoordinateReference) == "" {
		return Validation("coordinateReference", "不能为空")
	}
	if strings.TrimSpace(c.SpecificationRevision) == "" {
		return Validation("specificationRevision", "不能为空")
	}
	return nil
}

func ValidateControl(p ControlPoint) error {
	if strings.TrimSpace(p.Code) == "" {
		return Validation("code", "不能为空")
	}
	if math.IsNaN(p.Easting) || math.IsInf(p.Easting, 0) || math.Abs(p.Easting) > 100000000 {
		return Validation("easting", "坐标越界")
	}
	if math.IsNaN(p.Northing) || math.IsInf(p.Northing, 0) || math.Abs(p.Northing) > 100000000 {
		return Validation("northing", "坐标越界")
	}
	if math.IsNaN(p.Elevation) || math.IsInf(p.Elevation, 0) || p.Elevation < -12000 || p.Elevation > 12000 {
		return Validation("elevation", "高程越界")
	}
	if strings.TrimSpace(p.Source) == "" {
		return Validation("source", "不能为空")
	}
	if strings.TrimSpace(p.VerifiedBy) == "" {
		return Validation("verifiedBy", "不能为空")
	}
	if p.VerifiedAt.IsZero() || p.VerifiedAt.After(time.Now().Add(5*time.Minute)) {
		return Validation("verifiedAt", "必须为有效时间")
	}
	return nil
}

func ValidateObservation(o PipelineObservation) error {
	if strings.TrimSpace(o.SegmentCode) == "" {
		return Validation("segmentCode", "不能为空")
	}
	if strings.TrimSpace(o.UtilityType) == "" {
		return Validation("utilityType", "不能为空")
	}
	if o.StartPointID == "" || o.EndPointID == "" {
		return Validation("endpoints", "端点不能为空")
	}
	if o.StartPointID == o.EndPointID {
		return Validation("endPointId", "起止端点不能相同")
	}
	if o.BurialDepthMM < 0 || o.BurialDepthMM > 30000 {
		return Validation("burialDepthMm", "必须在 0 到 30000 之间")
	}
	if o.DiameterMM <= 0 || o.DiameterMM > 10000 {
		return Validation("diameterMm", "必须在 1 到 10000 之间")
	}
	if strings.TrimSpace(o.Material) == "" {
		return Validation("material", "不能为空")
	}
	if strings.TrimSpace(o.DetectionMethod) == "" {
		return Validation("detectionMethod", "不能为空")
	}
	if o.ObservedAt.IsZero() || o.ObservedAt.After(time.Now().Add(5*time.Minute)) {
		return Validation("observedAt", "必须为有效时间")
	}
	return nil
}

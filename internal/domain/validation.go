package domain

import (
	"fmt"
	"math"
	"strings"
	"time"
)

func normalizeRequiredSet(field string, values []string) ([]string, error) {
	if len(values) == 0 {
		return nil, Validation("empty_"+field, "%s 至少包含一项", field)
	}
	seen := map[string]bool{}
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			return nil, Validation("blank_"+field, "%s 不得包含空项", field)
		}
		if !seen[value] {
			seen[value] = true
			result = append(result, value)
		}
	}
	return result, nil
}

// MergeInvestigationDraft 只合并请求明确提供的字段，并统一执行草稿数据质量规则。
func MergeInvestigationDraft(inv *DeviationInvestigation, patch InvestigationDraftPatch) ([]string, error) {
	if inv.Status != InvestigationOpen {
		return nil, InvalidState("调查 %s 已闭合，不能修改草稿", inv.ID)
	}
	changed := make([]string, 0, 3)
	if patch.ImpactScope != nil {
		value := strings.TrimSpace(*patch.ImpactScope)
		if value == "" {
			return nil, Validation("blank_impact_scope", "impactScope 不能为空")
		}
		inv.ImpactScope = value
		changed = append(changed, "impactScope")
	}
	if patch.Hypotheses != nil {
		values, err := normalizeRequiredSet("hypotheses", *patch.Hypotheses)
		if err != nil {
			return nil, err
		}
		inv.Hypotheses = values
		changed = append(changed, "hypotheses")
	}
	if patch.EvidenceRefs != nil {
		values, err := normalizeRequiredSet("evidenceRefs", *patch.EvidenceRefs)
		if err != nil {
			return nil, err
		}
		inv.EvidenceRefs = values
		changed = append(changed, "evidenceRefs")
	}
	if len(changed) == 0 {
		return nil, Validation("empty_investigation_patch", "至少提供一个调查草稿字段")
	}
	return changed, nil
}

func ValidateCorrectiveAction(action CorrectiveAction, now time.Time, requireFutureDue bool) error {
	if strings.TrimSpace(action.ID) == "" || strings.TrimSpace(action.InvestigationID) == "" ||
		strings.TrimSpace(action.Description) == "" || strings.TrimSpace(action.Owner) == "" || action.DueAt.IsZero() {
		return Validation("incomplete_action", "纠正措施 id、investigationId、description、owner 和 dueAt 均不能为空")
	}
	if requireFutureDue && !action.DueAt.After(now) {
		return Validation("invalid_due_at", "纠正措施 %s 的 dueAt 必须晚于登记时间", action.ID)
	}
	return nil
}

var metricUnits = map[string]map[string]bool{
	"particle_0_5um":   {"particles/m3": true},
	"particle_5um":     {"particles/m3": true},
	"settle_plate":     {"cfu/plate": true},
	"airborne_microbe": {"cfu/m3": true},
}

func ValidateNewCampaign(c *MonitoringCampaign) error {
	if strings.TrimSpace(c.ID) == "" {
		return Validation("missing_id", "监测周期 id 不能为空")
	}
	if strings.TrimSpace(c.FacilityName) == "" {
		return Validation("missing_facility", "设施名称不能为空")
	}
	if len(c.Sites) == 0 {
		return Validation("missing_sites", "至少登记一个受控监测点")
	}
	seenID := map[string]bool{}
	seenPointMetric := map[string]bool{}
	for i := range c.Sites {
		s := &c.Sites[i]
		if s.CampaignID == "" {
			s.CampaignID = c.ID
		}
		if s.CampaignID != c.ID {
			return Validation("site_campaign_mismatch", "监测点 %s 不属于周期 %s", s.ID, c.ID)
		}
		if err := ValidateSite(*s); err != nil {
			return fmt.Errorf("监测点 %d: %w", i+1, err)
		}
		if seenID[s.ID] {
			return Validation("duplicate_site", "监测点 id %s 重复", s.ID)
		}
		key := strings.ToUpper(s.PointCode) + "|" + s.Metric
		if seenPointMetric[key] {
			return Validation("duplicate_point_metric", "监测点 %s 的指标 %s 重复", s.PointCode, s.Metric)
		}
		seenID[s.ID], seenPointMetric[key] = true, true
	}
	return nil
}

func ValidateSite(s ControlledSite) error {
	if strings.TrimSpace(s.ID) == "" || strings.TrimSpace(s.AreaName) == "" || strings.TrimSpace(s.PointCode) == "" {
		return Validation("incomplete_site", "id、区域名称和点位编码均不能为空")
	}
	if strings.TrimSpace(s.CleanlinessGrade) == "" {
		return Validation("missing_grade", "环境等级不能为空")
	}
	units, ok := metricUnits[s.Metric]
	if !ok {
		return Validation("unsupported_metric", "不支持指标 %s", s.Metric)
	}
	if !units[s.Unit] {
		return Validation("invalid_unit", "指标 %s 不支持单位 %s", s.Metric, s.Unit)
	}
	if math.IsNaN(s.AlertLimit) || math.IsInf(s.AlertLimit, 0) || s.AlertLimit <= 0 {
		return Validation("invalid_limit", "告警限值必须是有限正数")
	}
	return nil
}

func ValidateObservationValue(value float64) error {
	if math.IsNaN(value) || math.IsInf(value, 0) || value < 0 {
		return Validation("invalid_observed_value", "采样值必须是有限非负数")
	}
	return nil
}

func (c *MonitoringCampaign) Site(id string) (*ControlledSite, error) {
	for i := range c.Sites {
		if c.Sites[i].ID == id {
			return &c.Sites[i], nil
		}
	}
	return nil, NotFound("监测点", id)
}

func (c *MonitoringCampaign) Observation(id string) (*SampleObservation, error) {
	for i := range c.Observations {
		if c.Observations[i].ID == id {
			return &c.Observations[i], nil
		}
	}
	return nil, NotFound("采样结果", id)
}

func (c *MonitoringCampaign) Investigation(id string) (*DeviationInvestigation, error) {
	for i := range c.Investigations {
		if c.Investigations[i].ID == id {
			return &c.Investigations[i], nil
		}
	}
	return nil, NotFound("偏差调查", id)
}

func (c *MonitoringCampaign) Action(id string) (*CorrectiveAction, error) {
	for i := range c.Actions {
		if c.Actions[i].ID == id {
			return &c.Actions[i], nil
		}
	}
	return nil, NotFound("纠正措施", id)
}

package domain

import (
	"fmt"
	"strings"
	"time"
)

func NewCampaign(id, name, area, crs, spec string, now time.Time) (*SurveyCampaign, error) {
	c := &SurveyCampaign{ID: id, Name: strings.TrimSpace(name), SurveyArea: strings.TrimSpace(area), CoordinateReference: strings.TrimSpace(crs), SpecificationRevision: strings.TrimSpace(spec), State: StateDraft, Version: 1, CreatedAt: now.UTC(), UpdatedAt: now.UTC(), Controls: []ControlPoint{}, Observations: []PipelineObservation{}, Issues: []QualityIssue{}, Scans: []QualityScan{}, Rectifications: []Rectification{}, Reviews: []ReviewDecision{}}
	if err := ValidateCampaign(*c); err != nil {
		return nil, err
	}
	return c, nil
}

func (c *SurveyCampaign) ensureMutable() error {
	if c.State == StateFrozen || c.State == StateIssued {
		return Conflict("成果冻结后不可修改")
	}
	return nil
}

func (c *SurveyCampaign) bump(now time.Time) {
	c.Version++
	c.UpdatedAt = now.UTC()
}

func (c *SurveyCampaign) AddControl(p ControlPoint, now time.Time) error {
	if err := c.ensureMutable(); err != nil {
		return err
	}
	if c.State != StateDraft {
		return Conflict("只能在草稿状态登记控制点；基准变更必须形成新版本流程")
	}
	if err := ValidateControl(p); err != nil {
		return err
	}
	for _, existing := range c.Controls {
		if existing.ID == p.ID || strings.EqualFold(existing.Code, p.Code) {
			return Conflict("控制点编号重复")
		}
	}
	p.CampaignID = c.ID
	c.Controls = append(c.Controls, p)
	c.bump(now)
	return nil
}

func (c *SurveyCampaign) AmendControl(changeID, controlID, reason, actor string, replacement ControlPoint, now time.Time) error {
	if err := c.ensureMutable(); err != nil {
		return err
	}
	if c.State == StateUnderReview || c.State == StateApproved {
		return Conflict("复核中或已批准批次不能变更控制点")
	}
	if reason == "" {
		return Validation("reason", "控制点变更原因不能为空")
	}
	idx := -1
	for i := range c.Controls {
		if c.Controls[i].ID == controlID {
			idx = i
			break
		}
	}
	if idx < 0 {
		return NotFound("控制点")
	}
	before := c.Controls[idx]
	replacement.ID, replacement.CampaignID = before.ID, c.ID
	if err := ValidateControl(replacement); err != nil {
		return err
	}
	for i, other := range c.Controls {
		if i != idx && strings.EqualFold(other.Code, replacement.Code) {
			return Conflict("控制点编号重复")
		}
	}
	c.Controls[idx] = replacement
	c.ControlChanges = append(c.ControlChanges, ControlChange{ID: changeID, ControlID: controlID, Before: before, After: replacement, Reason: reason, Actor: actor, ChangedAt: now.UTC()})
	if c.State != StateDraft {
		c.State = StateBaselineLocked
	}
	c.bump(now)
	return nil
}

func (c *SurveyCampaign) LockBaseline(now time.Time) error {
	if c.State != StateDraft {
		return Conflict("当前状态不能锁定控制基准")
	}
	if len(c.Controls) < 2 {
		return Conflict("至少需要两个控制点")
	}
	c.State = StateBaselineLocked
	c.bump(now)
	return nil
}

func (c *SurveyCampaign) AddObservation(o PipelineObservation, now time.Time) error {
	if err := c.ensureMutable(); err != nil {
		return err
	}
	if c.State != StateBaselineLocked && c.State != StateQualityBlocked && c.State != StateReturned && c.State != StateReadyForReview {
		return Conflict("当前状态不能登记管段观测")
	}
	if err := ValidateObservation(o); err != nil {
		return err
	}
	if !c.HasControl(o.StartPointID) || !c.HasControl(o.EndPointID) {
		return Validation("endpoints", "引用了不存在的控制点")
	}
	for _, existing := range c.Observations {
		if existing.ID == o.ID || strings.EqualFold(existing.SegmentCode, o.SegmentCode) {
			return Conflict("管段 ID 或段号重复")
		}
	}
	o.CampaignID, o.Revision = c.ID, 1
	c.Observations = append(c.Observations, o)
	c.bump(now)
	return nil
}

// AddObservations 在修改聚合前完成整个候选集合的校验，因此任一输入失败都不会留下部分结果。
func (c *SurveyCampaign) AddObservations(candidates []PipelineObservation, now time.Time) error {
	if err := c.ensureMutable(); err != nil {
		return err
	}
	if c.State != StateBaselineLocked && c.State != StateQualityBlocked && c.State != StateReturned && c.State != StateReadyForReview {
		return Conflict("当前状态不能登记管段观测")
	}
	if len(candidates) < 1 || len(candidates) > 100 {
		return Validation("observations", "数量必须为 1 到 100 条")
	}
	knownIDs := make(map[string]struct{}, len(c.Observations)+len(candidates))
	knownCodes := make(map[string]struct{}, len(c.Observations)+len(candidates))
	for _, existing := range c.Observations {
		knownIDs[existing.ID] = struct{}{}
		knownCodes[strings.ToLower(existing.SegmentCode)] = struct{}{}
	}
	prepared := make([]PipelineObservation, len(candidates))
	for i, candidate := range candidates {
		if err := ValidateObservation(candidate); err != nil {
			return indexedValidation(i, err)
		}
		if !c.HasControl(candidate.StartPointID) {
			return Validation(fmt.Sprintf("observations[%d].startPointId", i), "控制点不存在或不属于当前批次")
		}
		if !c.HasControl(candidate.EndPointID) {
			return Validation(fmt.Sprintf("observations[%d].endPointId", i), "控制点不存在或不属于当前批次")
		}
		if _, exists := knownIDs[candidate.ID]; exists {
			return Validation(fmt.Sprintf("observations[%d].id", i), "观测标识重复")
		}
		codeKey := strings.ToLower(candidate.SegmentCode)
		if _, exists := knownCodes[codeKey]; exists {
			return Validation(fmt.Sprintf("observations[%d].segmentCode", i), "段号与请求内或批次既有观测重复（不区分大小写）")
		}
		knownIDs[candidate.ID], knownCodes[codeKey] = struct{}{}, struct{}{}
		candidate.CampaignID, candidate.Revision = c.ID, 1
		prepared[i] = candidate
	}
	c.Observations = append(c.Observations, prepared...)
	c.bump(now)
	return nil
}

func indexedValidation(index int, err error) error {
	if de, ok := err.(*Error); ok && de.Kind == ErrorValidation {
		return Validation(fmt.Sprintf("observations[%d].%s", index, de.Field), de.Message)
	}
	return err
}

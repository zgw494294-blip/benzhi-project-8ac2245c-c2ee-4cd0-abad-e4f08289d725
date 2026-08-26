package domain

import "time"

func (c *SurveyCampaign) ApplyScan(scan QualityScan, findings []QualityIssue, now time.Time) {
	active := make(map[string]QualityIssue, len(findings))
	for _, issue := range findings {
		active[issue.RuleCode+"|"+issue.ObjectRef] = issue
	}
	for i := range c.Issues {
		key := c.Issues[i].RuleCode + "|" + c.Issues[i].ObjectRef
		if _, found := active[key]; !found && c.Issues[i].Status == IssueOpen {
			c.Issues[i].Status = IssueResolved
			c.Issues[i].ScanID = scan.ID
			resolved := now.UTC()
			c.Issues[i].ResolvedAt = &resolved
		}
	}
	for _, finding := range findings {
		key := finding.RuleCode + "|" + finding.ObjectRef
		found := false
		for i := range c.Issues {
			if c.Issues[i].RuleCode+"|"+c.Issues[i].ObjectRef != key {
				continue
			}
			if c.Issues[i].Status == IssueResolved {
				c.Issues[i].Status, c.Issues[i].ResolvedAt, c.Issues[i].DetectedAt = IssueOpen, nil, now.UTC()
			}
			c.Issues[i].ScanID, c.Issues[i].Description, c.Issues[i].Severity = scan.ID, finding.Description, finding.Severity
			found = true
			break
		}
		if !found {
			c.Issues = append(c.Issues, finding)
		}
	}
	c.Scans = append(c.Scans, scan)
	if c.OpenBlockerCount() > 0 {
		c.State = StateQualityBlocked
	} else {
		c.State = StateReadyForReview
	}
	c.bump(now)
}

func (c *SurveyCampaign) ReviseObservation(issueID, rectificationID, note, actor string, replacement *PipelineObservation, now time.Time) error {
	if c.State != StateQualityBlocked && c.State != StateReturned && c.State != StateReadyForReview {
		return Conflict("当前状态不能整改")
	}
	issue := c.FindIssue(issueID)
	if issue == nil || issue.Status != IssueOpen {
		return Validation("issueId", "未找到未解决问题")
	}
	if note == "" {
		return Validation("note", "整改说明不能为空")
	}
	r := Rectification{ID: rectificationID, IssueID: issueID, Note: note, Actor: actor, SubmittedAt: now.UTC()}
	if replacement != nil {
		idx := c.ObservationIndex(replacement.ID)
		if idx < 0 {
			return Validation("observation.id", "观测不存在")
		}
		before := c.Observations[idx]
		replacement.CampaignID, replacement.Revision = c.ID, before.Revision+1
		if err := ValidateObservation(*replacement); err != nil {
			return err
		}
		if !c.HasControl(replacement.StartPointID) || !c.HasControl(replacement.EndPointID) {
			return Validation("observation.endpoints", "引用了不存在的控制点")
		}
		for i, other := range c.Observations {
			if i != idx && other.SegmentCode == replacement.SegmentCode {
				return Conflict("管段段号重复")
			}
		}
		c.Observations[idx] = *replacement
		after := *replacement
		r.Change = &ObservationChange{ObservationID: replacement.ID, Before: &before, After: &after}
	}
	c.Rectifications = append(c.Rectifications, r)
	c.bump(now)
	return nil
}

func (c *SurveyCampaign) SubmitReview(reviewer string, now time.Time) error {
	if c.State != StateReadyForReview {
		return Conflict("只有扫描通过的批次可以提交复核")
	}
	if c.OpenBlockerCount() != 0 || len(c.Scans) == 0 {
		return Conflict("仍有阻断问题或尚未扫描")
	}
	c.State = StateUnderReview
	c.bump(now)
	return nil
}

func (c *SurveyCampaign) DecideReview(id, reviewer, decision, reason string, now time.Time) error {
	if c.State != StateUnderReview {
		return Conflict("批次不在复核中")
	}
	if reviewer == "" {
		return Validation("reviewer", "不能为空")
	}
	if decision != "approve" && decision != "return" {
		return Validation("decision", "必须为 approve 或 return")
	}
	if decision == "return" && reason == "" {
		return Validation("reason", "退回必须填写理由")
	}
	last := c.Scans[len(c.Scans)-1]
	c.Reviews = append(c.Reviews, ReviewDecision{ID: id, Reviewer: reviewer, Decision: decision, Reason: reason, BoundScanID: last.ID, BoundVersion: c.Version, DecidedAt: now.UTC()})
	if decision == "approve" {
		c.State = StateApproved
	} else {
		c.State = StateReturned
	}
	c.bump(now)
	return nil
}

func (c *SurveyCampaign) Freeze(snapshotDigest, chainRoot string, now time.Time) error {
	if c.State != StateApproved {
		return Conflict("只有已批准批次可以冻结")
	}
	if c.OpenBlockerCount() != 0 {
		return Conflict("存在未解决阻断问题")
	}
	if snapshotDigest == "" || chainRoot == "" {
		return Validation("digest", "冻结摘要不能为空")
	}
	c.bump(now)
	c.State = StateFrozen
	c.Frozen = &FrozenSnapshot{CampaignID: c.ID, FrozenVersion: c.Version, SnapshotDigest: snapshotDigest, EventChainRoot: chainRoot, FrozenAt: now.UTC()}
	return nil
}

func (c *SurveyCampaign) IssueCredential(credential ReleaseCredential, now time.Time) error {
	if c.State != StateFrozen || c.Frozen == nil {
		return Conflict("只有冻结批次可以签发凭据")
	}
	if c.Credential != nil {
		return Conflict("准入凭据已经签发")
	}
	if credential.CampaignID != c.ID || credential.FrozenVersion != c.Frozen.FrozenVersion || credential.SnapshotDigest != c.Frozen.SnapshotDigest || credential.EventChainRoot != c.Frozen.EventChainRoot {
		return Validation("credential", "凭据与冻结快照不一致")
	}
	c.Credential = &credential
	c.State = StateIssued
	c.bump(now)
	return nil
}

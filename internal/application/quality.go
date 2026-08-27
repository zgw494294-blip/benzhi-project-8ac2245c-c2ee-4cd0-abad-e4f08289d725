package application

import (
	"context"
	"fmt"
	"net/http"

	"subsurface-survey-gate/internal/domain"
	"subsurface-survey-gate/internal/quality"
)

func (s *Service) RunScan(ctx context.Context, campaignID string, cmd RunScan) (Result, error) {
	if err := validateMetadata(cmd.Metadata, false); err != nil {
		return Result{}, err
	}
	fp, replay, err := s.prepare(ctx, campaignID, cmd.IdempotencyKey, cmd)
	if err != nil {
		return Result{}, err
	}
	if replay != nil {
		return *replay, nil
	}
	c, err := s.store.Load(ctx, campaignID)
	if err != nil {
		return Result{}, err
	}
	if c.State != domain.StateBaselineLocked && c.State != domain.StateQualityBlocked && c.State != domain.StateReturned && c.State != domain.StateReadyForReview {
		return Result{}, domain.Conflict("当前状态不能执行质量扫描")
	}
	now := s.now().UTC()
	previousState := c.State
	result := s.scanner.Scan(c)
	scanID := "scan_" + domain.Digest(fmt.Sprintf("%s|%d|%s", c.ID, c.Version, result.InputDigest))[:24]
	issues := make([]domain.QualityIssue, 0, len(result.Findings))
	snapshotFindings := make([]domain.ScanFinding, 0, len(result.Findings))
	blockers := 0
	for _, f := range result.Findings {
		if f.Severity == domain.SeverityBlocker {
			blockers++
		}
		id := domain.IssueIDForFinding(f.RuleCode, f.ObjectRef)
		issues = append(issues, domain.QualityIssue{ID: id, CampaignID: c.ID, ScanID: scanID, RuleCode: f.RuleCode, Severity: f.Severity, ObjectRef: f.ObjectRef, Description: f.Description, Status: domain.IssueOpen, DetectedAt: now})
		snapshotFindings = append(snapshotFindings, domain.ScanFinding{Key: domain.FindingKey(f.RuleCode, f.ObjectRef), RuleCode: f.RuleCode, Severity: f.Severity, ObjectRef: f.ObjectRef, Description: f.Description})
	}
	scan := domain.QualityScan{ID: scanID, CampaignID: c.ID, RuleSetVersion: result.RuleSetVersion, InputDigest: result.InputDigest, IssueCount: len(issues), BlockerCount: blockers, ScannedAt: now, Findings: snapshotFindings}
	c.ApplyScan(scan, issues, now)
	e := domain.NewStateEvent(c, newID("evt"), "quality.scanned", cmd.Actor, now, previousState, map[string]any{"scan": scan, "findings": issues})
	return s.commit(ctx, campaignID, cmd.ExpectedVersion, c, []domain.Event{e}, cmd.IdempotencyKey, fp, http.StatusOK, c)
}

func (s *Service) CompareScans(ctx context.Context, campaignID, baseScanID, targetScanID string) (domain.ScanDifferenceReport, error) {
	if baseScanID == "" {
		return domain.ScanDifferenceReport{}, domain.QueryError("baseScanId", "不能为空")
	}
	if targetScanID == "" {
		return domain.ScanDifferenceReport{}, domain.QueryError("targetScanId", "不能为空")
	}
	c, err := s.store.Load(ctx, campaignID)
	if err != nil {
		return domain.ScanDifferenceReport{}, err
	}
	baseIndex, targetIndex := c.ScanIndex(baseScanID), c.ScanIndex(targetScanID)
	if baseIndex < 0 {
		return domain.ScanDifferenceReport{}, domain.QueryError("baseScanId", "扫描不存在或不属于当前批次")
	}
	if targetIndex < 0 {
		return domain.ScanDifferenceReport{}, domain.QueryError("targetScanId", "扫描不存在或不属于当前批次")
	}
	if baseIndex == targetIndex {
		return domain.ScanDifferenceReport{}, domain.QueryError("targetScanId", "不能比较同一次扫描")
	}
	if baseIndex > targetIndex {
		return domain.ScanDifferenceReport{}, domain.QueryError("targetScanId", "扫描时间倒置，目标扫描必须晚于基准扫描")
	}
	report := quality.CompareScans(c.Scans[baseIndex], c.Scans[targetIndex])
	report.CampaignID, report.Version = c.ID, c.Version
	for i := range report.Resolved {
		item := &report.Resolved[i]
		issueID := domain.IssueIDForFinding(item.Finding.RuleCode, item.Finding.ObjectRef)
		if rectification := c.LatestRectificationBefore(issueID, c.Scans[targetIndex].ScannedAt); rectification != nil {
			item.ResolutionType, item.Rectification = "rectification", rectification
		} else {
			item.ResolutionType = "rule_result"
		}
	}
	return report, nil
}

func (s *Service) SubmitRectification(ctx context.Context, campaignID string, cmd SubmitRectification) (Result, error) {
	if err := validateMetadata(cmd.Metadata, false); err != nil {
		return Result{}, err
	}
	fp, replay, err := s.prepare(ctx, campaignID, cmd.IdempotencyKey, cmd)
	if err != nil {
		return Result{}, err
	}
	if replay != nil {
		return *replay, nil
	}
	c, err := s.store.Load(ctx, campaignID)
	if err != nil {
		return Result{}, err
	}
	now := s.now().UTC()
	id := newID("fix")
	if err := c.ReviseObservation(cmd.IssueID, id, cmd.Note, cmd.Actor, cmd.RevisedObservation, now); err != nil {
		return Result{}, err
	}
	r := c.Rectifications[len(c.Rectifications)-1]
	e := domain.NewEvent(c, newID("evt"), "rectification.submitted", cmd.Actor, now, r)
	return s.commit(ctx, campaignID, cmd.ExpectedVersion, c, []domain.Event{e}, cmd.IdempotencyKey, fp, http.StatusOK, c)
}

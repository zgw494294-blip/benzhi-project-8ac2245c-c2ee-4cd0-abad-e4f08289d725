package application

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"subsurface-survey-gate/internal/domain"
	"subsurface-survey-gate/internal/eventstore"
	"subsurface-survey-gate/internal/quality"
)

func TestExtendedQualityFlow(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	store, err := eventstore.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 8, 26, 10, 0, 0, 0, time.UTC)
	service := NewService(store, quality.NewScanner())
	service.now = func() time.Time { return now }

	result, err := service.CreateCampaign(ctx, CreateCampaign{Metadata: Metadata{IdempotencyKey: "create"}, Name: "测试批次", SurveyArea: "A区", CoordinateReference: "CGCS2000", SpecificationRevision: "r1", Actor: "负责人"})
	created := mustCampaignResult(t, result, err)
	controlIDs := make([]string, 0, 3)
	for i, code := range []string{"P1", "P2", "P3"} {
		result, err := service.AddControl(ctx, created.ID, AddControl{Metadata: Metadata{ExpectedVersion: created.Version, IdempotencyKey: "control-" + code}, Code: code, Easting: float64(i * 100), Northing: float64(i * 50), Elevation: 10, Source: "GNSS", VerifiedBy: "复核员", VerifiedAt: now.Add(-time.Minute), Actor: "负责人"})
		created = mustCampaignResult(t, result, err)
		controlIDs = append(controlIDs, created.Controls[len(created.Controls)-1].ID)
	}
	readiness, err := service.BaselineReadiness(ctx, created.ID)
	if err != nil || !readiness.ReadyToLock || readiness.ReadinessDigest == "" || readiness.Version != created.Version {
		t.Fatalf("预检异常: %#v, %v", readiness, err)
	}
	result, err = service.LockBaseline(ctx, created.ID, LockBaseline{Metadata: Metadata{ExpectedVersion: created.Version, IdempotencyKey: "lock"}, Actor: "负责人"})
	locked := mustCampaignResult(t, result, err)

	invalid := AddObservationBatch{Metadata: Metadata{ExpectedVersion: locked.Version, IdempotencyKey: "batch-invalid"}, Actor: "探测员", Observations: []ObservationInput{
		{SegmentCode: "S1", UtilityType: "给水", StartPointID: controlIDs[0], EndPointID: controlIDs[1], BurialDepthMM: 1000, DiameterMM: 100, Material: "钢", DetectionMethod: "电磁", ObservedAt: now.Add(-time.Minute)},
		{SegmentCode: "S2", UtilityType: "给水", StartPointID: controlIDs[1], EndPointID: "outside", BurialDepthMM: 6000, DiameterMM: 100, Material: "钢", DetectionMethod: "电磁", ObservedAt: now.Add(-time.Minute)},
	}}
	if _, err := service.AddObservationBatch(ctx, locked.ID, invalid); err == nil {
		t.Fatal("含悬空端点的批量请求不应成功")
	}
	afterInvalid, _ := store.Load(ctx, locked.ID)
	if afterInvalid.Version != locked.Version || len(afterInvalid.Observations) != 0 {
		t.Fatal("失败批量请求产生了部分写入")
	}

	valid := invalid
	valid.IdempotencyKey = "batch-valid"
	valid.Observations[1].EndPointID = controlIDs[2]
	batchResult, err := service.AddObservationBatch(ctx, locked.ID, valid)
	if err != nil {
		t.Fatal(err)
	}
	var batch ObservationBatchResult
	if err := json.Unmarshal(batchResult.Body, &batch); err != nil {
		t.Fatal(err)
	}
	if batch.Version != locked.Version+1 || len(batch.Observations) != 2 || batch.Observations[0].Revision != 1 || batch.Observations[1].Revision != 1 {
		t.Fatalf("批量登记结果异常: %#v", batch)
	}
	if replay, err := service.AddObservationBatch(ctx, locked.ID, valid); err != nil || !replay.Replayed {
		t.Fatalf("批量登记幂等重放失败: %v", err)
	}

	result, err = service.RunScan(ctx, locked.ID, RunScan{Metadata: Metadata{ExpectedVersion: batch.Version, IdempotencyKey: "scan-1"}, Actor: "质检员"})
	scanned := mustCampaignResult(t, result, err)
	baseScan := scanned.LatestScan()
	issues, err := service.Issues(ctx, locked.ID, IssueQuery{PageSize: 1})
	if err != nil || len(issues.Items) != 1 || issues.Summary.OpenCount == 0 {
		t.Fatalf("问题清单异常: %#v, %v", issues, err)
	}
	var issue *domain.QualityIssue
	for i := range scanned.Issues {
		if scanned.Issues[i].RuleCode == "DEPTH_JUMP" {
			issue = &scanned.Issues[i]
			break
		}
	}
	if issue == nil {
		t.Fatal("未产生预期埋深突变问题")
	}
	revised := batch.Observations[1]
	revised.BurialDepthMM = 1200
	result, err = service.SubmitRectification(ctx, locked.ID, SubmitRectification{Metadata: Metadata{ExpectedVersion: scanned.Version, IdempotencyKey: "fix"}, IssueID: issue.ID, Note: "复测修正埋深", Actor: "探测员", RevisedObservation: &revised})
	rectified := mustCampaignResult(t, result, err)
	result, err = service.RunScan(ctx, locked.ID, RunScan{Metadata: Metadata{ExpectedVersion: rectified.Version, IdempotencyKey: "scan-2"}, Actor: "质检员"})
	target := mustCampaignResult(t, result, err)
	report, err := service.CompareScans(ctx, locked.ID, baseScan.ID, target.LatestScan().ID)
	if err != nil || report.Summary.Resolved == 0 || report.Resolved[0].ResolutionType != "rectification" || report.Resolved[0].Rectification == nil {
		t.Fatalf("复扫差异异常: %#v, %v", report, err)
	}
	timeline, err := service.AuditTimeline(ctx, locked.ID, AuditQuery{Limit: 100})
	if err != nil || !timeline.ChainValidation.Valid || timeline.ChainRoot == "" || len(timeline.Events) != int(timeline.ChainValidation.ValidatedThrough) {
		t.Fatalf("审计时间线异常: %#v, %v", timeline, err)
	}
	firstPage, err := service.AuditTimeline(ctx, locked.ID, AuditQuery{Limit: 2})
	if err != nil || firstPage.NextCursor == "" {
		t.Fatalf("审计分页起始页异常: %#v, %v", firstPage, err)
	}
	reopenedStore, err := eventstore.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	reopened := NewService(reopenedStore, quality.NewScanner())
	secondPage, err := reopened.AuditTimeline(ctx, locked.ID, AuditQuery{Limit: 2, Cursor: firstPage.NextCursor})
	if err != nil || len(secondPage.Events) == 0 || secondPage.Events[0].Sequence <= firstPage.Events[len(firstPage.Events)-1].Sequence {
		t.Fatalf("重启后审计分页异常: %#v, %v", secondPage, err)
	}
	recoveredReport, err := reopened.CompareScans(ctx, locked.ID, baseScan.ID, target.LatestScan().ID)
	if err != nil || domain.Digest(recoveredReport) != domain.Digest(report) {
		t.Fatalf("重启后扫描差异不一致: %#v, %v", recoveredReport, err)
	}
}

func mustCampaignResult(t *testing.T, result Result, err error) *domain.SurveyCampaign {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
	var campaign domain.SurveyCampaign
	if err := json.Unmarshal(result.Body, &campaign); err != nil {
		t.Fatal(err)
	}
	return &campaign
}

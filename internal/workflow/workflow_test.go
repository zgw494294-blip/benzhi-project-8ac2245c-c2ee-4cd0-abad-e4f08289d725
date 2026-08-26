package workflow

import (
	"context"
	"fmt"
	"testing"
	"time"

	"cleanroom-release-go/internal/domain"
	"cleanroom-release-go/internal/ledger"
)

type flowHarness struct {
	t        *testing.T
	service  *Service
	ctx      context.Context
	campaign *domain.MonitoringCampaign
	key      int
}

func newFlow(t *testing.T) *flowHarness {
	store, err := ledger.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	return &flowHarness{t: t, service: New(store, "test-secret"), ctx: context.Background()}
}
func (h *flowHarness) meta() WriteMeta {
	h.key++
	version := int64(0)
	if h.campaign != nil {
		version = h.campaign.Version
	}
	return WriteMeta{ExpectedVersion: version, IdempotencyKey: fmt.Sprintf("key-%d", h.key), Actor: "测试员"}
}
func (h *flowHarness) set(c *domain.MonitoringCampaign, err error) *domain.MonitoringCampaign {
	h.t.Helper()
	if err != nil {
		h.t.Fatal(err)
	}
	h.campaign = c
	return c
}

func TestDeviationReworkFreezeAndCredential(t *testing.T) {
	h := newFlow(t)
	m := h.meta()
	h.set(h.service.CreateCampaign(h.ctx, CreateCampaignCommand{WriteMeta: m, ID: "c1", FacilityName: "无菌厂房", Sites: []domain.ControlledSite{{ID: "s1", AreaName: "灌装间", PointCode: "P1", CleanlinessGrade: "B", Metric: "settle_plate", Unit: "cfu/plate", AlertLimit: 5}}}))
	m = h.meta()
	h.set(h.service.LockPlan(h.ctx, "c1", LockPlanCommand{WriteMeta: m, Reviewer: "复核员", PlannedRounds: 1}))
	m = h.meta()
	h.set(h.service.RecordPlannedObservation(h.ctx, "c1", RecordObservationCommand{WriteMeta: m, ID: "o1", SiteID: "s1", RoundNumber: 1, ObservedValue: 6, Unit: "cfu/plate"}))
	if h.campaign.Status != domain.StatusInvestigation {
		t.Fatal(h.campaign.Status)
	}
	h.closeInvestigation("inv-o1", "初始根因")
	h.completeAction("a1", "inv-o1")
	m = h.meta()
	h.set(h.service.BeginVerification(h.ctx, "c1", BeginVerificationCommand{WriteMeta: m}))
	m = h.meta()
	h.set(h.service.RecordVerificationObservation(h.ctx, "c1", RecordObservationCommand{WriteMeta: m, ID: "v1", SiteID: "s1", RoundNumber: 1, ObservedValue: 7, Unit: "cfu/plate"}))
	if h.campaign.Status != domain.StatusInvestigation {
		t.Fatal("再次超限未退回调查")
	}
	h.closeInvestigation("inv-v1", "消毒时间不足")
	h.completeAction("a2", "inv-v1")
	m = h.meta()
	h.set(h.service.BeginVerification(h.ctx, "c1", BeginVerificationCommand{WriteMeta: m}))
	m = h.meta()
	h.set(h.service.RecordVerificationObservation(h.ctx, "c1", RecordObservationCommand{WriteMeta: m, ID: "v2", SiteID: "s1", RoundNumber: 2, ObservedValue: 1, Unit: "cfu/plate"}))
	if h.campaign.Status != domain.StatusVerificationPassed {
		t.Fatal(h.campaign.Status)
	}
	m = h.meta()
	h.set(h.service.Review(h.ctx, "c1", ReviewCommand{WriteMeta: m, Decision: "reject", Comment: "补充说明"}))
	if h.campaign.Status != domain.StatusVerificationPassed {
		t.Fatal("退回应保持待审核状态")
	}
	m = h.meta()
	h.set(h.service.Review(h.ctx, "c1", ReviewCommand{WriteMeta: m, Decision: "approve", Comment: "同意放行"}))
	if h.campaign.Status != domain.StatusFrozen || h.campaign.FrozenDigest == "" {
		t.Fatal("审核通过后未冻结")
	}
	m = h.meta()
	credential, err := h.service.IssueCredential(h.ctx, "c1", IssueCredentialCommand{WriteMeta: m, ID: "cred-1", IssuedBy: "审核员"})
	if err != nil {
		t.Fatal(err)
	}
	if credential.SnapshotDigest != h.campaign.FrozenDigest {
		t.Fatal("凭据摘要不匹配")
	}
	verified, err := h.service.VerifyCredential(h.ctx, "cred-1")
	if err != nil {
		t.Fatal(err)
	}
	if !verified.Valid {
		t.Fatal(verified.Reason)
	}
	m = h.meta()
	if _, err := h.service.LockPlan(h.ctx, "c1", LockPlanCommand{WriteMeta: m, Reviewer: "其他人", PlannedRounds: 2}); err == nil {
		t.Fatal("冻结后应禁止业务修改")
	}
}

func (h *flowHarness) closeInvestigation(id, root string) {
	h.t.Helper()
	m := h.meta()
	h.set(h.service.ConcludeInvestigation(h.ctx, "c1", id, ConcludeInvestigationCommand{WriteMeta: m, ImpactScope: "当班区域", Hypotheses: []string{"设备或操作"}, EvidenceRefs: []string{"evidence://record"}, RootCause: root}))
}
func (h *flowHarness) completeAction(id, inv string) {
	h.t.Helper()
	m := h.meta()
	h.set(h.service.AddCorrectiveAction(h.ctx, "c1", AddCorrectiveActionCommand{WriteMeta: m, ID: id, InvestigationID: inv, Description: "完成根因纠正", Owner: "负责人", DueAt: time.Now().Add(time.Hour)}))
	m = h.meta()
	h.set(h.service.CompleteCorrectiveAction(h.ctx, "c1", id, CompleteCorrectiveActionCommand{WriteMeta: m, CompletionEvidence: "evidence://done"}))
}

func TestStaleVersionAndIncompleteSamplingAreRejected(t *testing.T) {
	h := newFlow(t)
	m := h.meta()
	h.set(h.service.CreateCampaign(h.ctx, CreateCampaignCommand{WriteMeta: m, ID: "c1", FacilityName: "厂房", Sites: []domain.ControlledSite{{ID: "s1", AreaName: "A", PointCode: "P1", CleanlinessGrade: "C", Metric: "settle_plate", Unit: "cfu/plate", AlertLimit: 2}, {ID: "s2", AreaName: "A", PointCode: "P2", CleanlinessGrade: "C", Metric: "settle_plate", Unit: "cfu/plate", AlertLimit: 2}}}))
	m = h.meta()
	m.ExpectedVersion = 0
	if _, err := h.service.LockPlan(h.ctx, "c1", LockPlanCommand{WriteMeta: m, Reviewer: "r", PlannedRounds: 1}); err == nil {
		t.Fatal("陈旧版本应冲突")
	}
}

func TestInvestigationDraftAndAtomicCorrectionBatches(t *testing.T) {
	h := newFlow(t)
	fixed := time.Date(2026, 8, 27, 2, 0, 0, 0, time.UTC)
	h.service.now = func() time.Time { return fixed }
	m := h.meta()
	h.set(h.service.CreateCampaign(h.ctx, CreateCampaignCommand{WriteMeta: m, ID: "c1", FacilityName: "厂房", Sites: []domain.ControlledSite{{ID: "s1", AreaName: "A", PointCode: "P1", CleanlinessGrade: "C", Metric: "settle_plate", Unit: "cfu/plate", AlertLimit: 2}}}))
	m = h.meta()
	h.set(h.service.LockPlan(h.ctx, "c1", LockPlanCommand{WriteMeta: m, Reviewer: "r", PlannedRounds: 1}))
	m = h.meta()
	h.set(h.service.RecordPlannedObservation(h.ctx, "c1", RecordObservationCommand{WriteMeta: m, ID: "o1", SiteID: "s1", RoundNumber: 1, ObservedValue: 3, Unit: "cfu/plate"}))
	impact := " 当班区域 "
	hypotheses := []string{"设备", "设备"}
	evidence := []string{" evidence://one "}
	before := h.campaign.Version
	m = h.meta()
	h.set(h.service.UpdateInvestigationDraft(h.ctx, "c1", "inv-o1", UpdateInvestigationDraftCommand{WriteMeta: m, ImpactScope: &impact, Hypotheses: &hypotheses, EvidenceRefs: &evidence}))
	inv, _ := h.campaign.Investigation("inv-o1")
	if h.campaign.Version != before+1 || inv.ImpactScope != "当班区域" || len(inv.Hypotheses) != 1 || inv.EvidenceRefs[0] != "evidence://one" {
		t.Fatalf("草稿规范化异常: %+v", inv)
	}
	preflight, err := h.service.InvestigationPreflight(h.ctx, "c1", "inv-o1")
	if err != nil || preflight.CanConclude || len(preflight.MissingFields) != 1 || preflight.MissingFields[0] != "rootCause" {
		t.Fatalf("调查预检异常: %+v, %v", preflight, err)
	}
	m = h.meta()
	h.set(h.service.ConcludeInvestigation(h.ctx, "c1", "inv-o1", ConcludeInvestigationCommand{WriteMeta: m, RootCause: "设备老化"}))
	m = h.meta()
	h.set(h.service.BatchAddCorrectiveActions(h.ctx, "c1", BatchAddCorrectiveActionsCommand{WriteMeta: m, Actions: []CorrectiveActionInput{
		{ID: "a1", InvestigationID: "inv-o1", Description: "更换", Owner: "甲", DueAt: fixed.Add(time.Hour)},
		{ID: "a2", InvestigationID: "inv-o1", Description: "复核", Owner: "乙", DueAt: fixed.Add(2 * time.Hour)},
	}}))
	version := h.campaign.Version
	m = h.meta()
	_, err = h.service.BatchCompleteCorrectiveActions(h.ctx, "c1", BatchCompleteCorrectiveActionsCommand{WriteMeta: m, Actions: []CorrectiveActionCompletionInput{{ID: "a1", CompletionEvidence: "evidence://a1"}, {ID: "a2"}}})
	if err == nil {
		t.Fatal("空完成证据应使整批失败")
	}
	current, err := h.service.GetCampaign(h.ctx, "c1")
	if err != nil {
		t.Fatal(err)
	}
	if current.Version != version || current.Actions[0].CompletedAt != nil || current.Actions[1].CompletedAt != nil {
		t.Fatal("失败批次出现部分落账")
	}
	h.campaign = current
	m = h.meta()
	h.set(h.service.BatchCompleteCorrectiveActions(h.ctx, "c1", BatchCompleteCorrectiveActionsCommand{WriteMeta: m, Actions: []CorrectiveActionCompletionInput{{ID: "a1", CompletionEvidence: "evidence://a1"}, {ID: "a2", CompletionEvidence: "evidence://a2"}}}))
	if h.campaign.Version != version+1 || !h.campaign.Actions[0].CompletedAt.Equal(*h.campaign.Actions[1].CompletedAt) {
		t.Fatal("成功批次未使用同一业务时间或单版本增量")
	}
}

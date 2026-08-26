package assessment

import (
	"testing"
	"time"

	"cleanroom-release-go/internal/domain"
)

func TestAssessObservationBoundaryAndUnit(t *testing.T) {
	site := domain.ControlledSite{ID: "s1", Metric: "settle_plate", Unit: "cfu/plate", AlertLimit: 5}
	decision, err := AssessObservation(site, 5, "cfu/plate")
	if err != nil {
		t.Fatal(err)
	}
	if decision.Verdict != domain.VerdictNormal {
		t.Fatalf("限值边界应为正常，实际 %s", decision.Verdict)
	}
	decision, err = AssessObservation(site, 5.1, "cfu/plate")
	if err != nil {
		t.Fatal(err)
	}
	if decision.Verdict != domain.VerdictAlert {
		t.Fatalf("超过限值应为超限，实际 %s", decision.Verdict)
	}
	if _, err = AssessObservation(site, 1, "cfu/m3"); err == nil {
		t.Fatal("单位不匹配应返回错误")
	}
}

func TestPlannedAndVerificationCompleteness(t *testing.T) {
	c := &domain.MonitoringCampaign{PlannedRounds: 2, VerificationRound: 1, Sites: []domain.ControlledSite{{ID: "a", Metric: "airborne_microbe"}, {ID: "b", Metric: "particle_0_5um"}}, Observations: []domain.SampleObservation{
		{ID: "p1", SiteID: "a", RoundKind: domain.RoundPlanned, RoundNumber: 1, Verdict: domain.VerdictNormal},
		{ID: "p2", SiteID: "b", RoundKind: domain.RoundPlanned, RoundNumber: 1, Verdict: domain.VerdictNormal},
		{ID: "v1", SiteID: "a", RoundKind: domain.RoundVerification, RoundNumber: 1, Verdict: domain.VerdictNormal},
	}}
	planned := PlannedSamplingCompleteness(c)
	if planned.Complete || len(planned.Missing) != 2 {
		t.Fatalf("应缺少第二轮两个点位: %+v", planned)
	}
	verification := AssessVerification(c, 1)
	if verification.Complete || verification.Passed || len(verification.MissingSites) != 1 {
		t.Fatalf("验证完整性结论错误: %+v", verification)
	}
}

func TestInvestigationAndCorrectionCompleteness(t *testing.T) {
	inv := domain.DeviationInvestigation{ID: "i1", ImpactScope: "区域", Hypotheses: []string{"假设"}, EvidenceRefs: []string{"evidence://1"}}
	check := InvestigationCompleteness(inv)
	if check.Complete || len(check.Missing) != 1 || check.Missing[0] != "rootCause" {
		t.Fatalf("调查缺失项错误: %+v", check)
	}
	now := time.Now()
	inv.RootCause = "根因"
	inv.Status = domain.InvestigationConcluded
	c := &domain.MonitoringCampaign{Investigations: []domain.DeviationInvestigation{inv}, Actions: []domain.CorrectiveAction{{ID: "a1", InvestigationID: "i1", Description: "修复", Owner: "负责人", CompletedAt: &now, CompletionEvidence: "evidence://done"}}}
	if result := CorrectionCompleteness(c); !result.Complete {
		t.Fatalf("纠正措施应完整: %+v", result)
	}
}

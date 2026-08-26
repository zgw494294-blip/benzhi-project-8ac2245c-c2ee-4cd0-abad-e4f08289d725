package assessment

import (
	"testing"
	"time"

	"cleanroom-release-go/internal/domain"
)

func TestSamplingCoverageMatrixAndAlertWorkItems(t *testing.T) {
	c := &domain.MonitoringCampaign{ID: "c1", Version: 7, PlannedRounds: 2, PlanDigest: "locked", Sites: []domain.ControlledSite{
		{ID: "s1", AreaName: "A", PointCode: "P1", Metric: "settle_plate"},
		{ID: "s2", AreaName: "A", PointCode: "P2", Metric: "settle_plate"},
		{ID: "s3", AreaName: "B", PointCode: "P3", Metric: "airborne_microbe"},
	}}
	c.Observations = []domain.SampleObservation{
		{ID: "o11", CampaignID: "c1", SiteID: "s1", RoundKind: domain.RoundPlanned, RoundNumber: 1, Verdict: domain.VerdictNormal},
		{ID: "o12", CampaignID: "c1", SiteID: "s2", RoundKind: domain.RoundPlanned, RoundNumber: 1, Verdict: domain.VerdictAlert},
		{ID: "o13", CampaignID: "c1", SiteID: "s3", RoundKind: domain.RoundPlanned, RoundNumber: 1, Verdict: domain.VerdictNormal},
		{ID: "o21", CampaignID: "c1", SiteID: "s1", RoundKind: domain.RoundPlanned, RoundNumber: 2, Verdict: domain.VerdictNormal},
	}
	c.Investigations = []domain.DeviationInvestigation{{ID: "inv-o12", CampaignID: "c1", ObservationID: "o12", Status: domain.InvestigationOpen}}
	result, err := SamplingProgress(c, domain.SamplingProgressFilter{})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Cells) != 6 || result.Rounds[0].Recorded != 3 || result.Rounds[1].Missing != 2 || result.CompletionRatio != 4.0/6.0 {
		t.Fatalf("覆盖统计异常: %+v", result)
	}
	if len(result.AlertWorkItems) != 1 || result.AlertWorkItems[0].InvestigationID != "inv-o12" {
		t.Fatalf("超限清单异常: %+v", result.AlertWorkItems)
	}
}

func TestVerificationComparisonKeepsRoundsSeparate(t *testing.T) {
	c := &domain.MonitoringCampaign{ID: "c1", Version: 12, Status: domain.StatusVerificationPassed, VerificationRound: 2,
		Sites: []domain.ControlledSite{{ID: "s1", AreaName: "A", PointCode: "P1", Metric: "settle_plate", AlertLimit: 5}},
		Observations: []domain.SampleObservation{
			{ID: "v1", CampaignID: "c1", SiteID: "s1", RoundKind: domain.RoundVerification, RoundNumber: 1, ObservedValue: 7, Unit: "cfu/plate", Verdict: domain.VerdictAlert},
			{ID: "v2", CampaignID: "c1", SiteID: "s1", RoundKind: domain.RoundVerification, RoundNumber: 2, ObservedValue: 1, Unit: "cfu/plate", Verdict: domain.VerdictNormal},
		}, Investigations: []domain.DeviationInvestigation{{ID: "inv-v1", CampaignID: "c1", ObservationID: "v1", Status: domain.InvestigationConcluded}},
	}
	result, err := VerificationRounds(c, 1, 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Sites) != 1 || len(result.Sites[0].Rounds) != 2 || result.Sites[0].Rounds[0].InvestigationID != "inv-v1" || result.Rounds[0].Conclusion != "failed" || result.Rounds[1].Conclusion != "passed" {
		t.Fatalf("验证对比异常: %+v", result)
	}
	preflightCampaign := &domain.MonitoringCampaign{ID: "c2", Status: domain.StatusCorrection, Investigations: []domain.DeviationInvestigation{{ID: "inv", Status: domain.InvestigationConcluded}}, Actions: []domain.CorrectiveAction{{ID: "a", InvestigationID: "inv", DueAt: time.Now().Add(time.Hour)}}}
	preflight := VerificationPreflight(preflightCampaign)
	if preflight.CanStart || len(preflight.Blockers) != 1 || preflight.Blockers[0].ActionID != "a" {
		t.Fatalf("开轮预检异常: %+v", preflight)
	}
}

package assessment

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"cleanroom-release-go/internal/domain"
)

func orderedSites(c *domain.MonitoringCampaign) []domain.ControlledSite {
	sites := append([]domain.ControlledSite(nil), c.Sites...)
	sort.Slice(sites, func(i, j int) bool {
		a, b := sites[i], sites[j]
		if a.AreaName != b.AreaName {
			return a.AreaName < b.AreaName
		}
		if a.PointCode != b.PointCode {
			return a.PointCode < b.PointCode
		}
		if a.Metric != b.Metric {
			return a.Metric < b.Metric
		}
		return a.ID < b.ID
	})
	return sites
}

func validateProgressFilter(c *domain.MonitoringCampaign, filter domain.SamplingProgressFilter) error {
	if filter.RoundNumber < 0 || filter.RoundNumber > c.PlannedRounds {
		return domain.Validation("invalid_round", "计划轮次必须在 1 到 %d 之间", c.PlannedRounds)
	}
	if filter.Metric != "" {
		found := false
		for _, site := range c.Sites {
			if site.Metric == filter.Metric {
				found = true
				break
			}
		}
		if !found {
			return domain.Validation("unsupported_metric", "方案中不存在指标 %s", filter.Metric)
		}
	}
	return nil
}

func SamplingProgress(c *domain.MonitoringCampaign, filter domain.SamplingProgressFilter) (domain.SamplingProgress, error) {
	result := domain.SamplingProgress{CampaignID: c.ID, Version: c.Version, Cells: []domain.SamplingCoverageCell{}, Rounds: []domain.SamplingRoundSummary{}, AlertWorkItems: []domain.AlertWorkItem{}}
	if c.PlannedRounds < 1 || c.PlanDigest == "" {
		result.Reason = "采样方案尚未锁定，无法评估计划覆盖情况"
		return result, nil
	}
	if err := validateProgressFilter(c, filter); err != nil {
		return result, err
	}
	sites := orderedSites(c)
	siteByID := make(map[string]domain.ControlledSite, len(sites))
	for _, site := range sites {
		if _, exists := siteByID[site.ID]; exists {
			return result, domain.Conflict("data_consistency_error", "投影中存在重复监测点 %s", site.ID)
		}
		siteByID[site.ID] = site
	}
	observations := map[string]domain.SampleObservation{}
	for _, observation := range c.Observations {
		if observation.RoundKind != domain.RoundPlanned {
			continue
		}
		if observation.CampaignID != c.ID {
			return result, domain.Conflict("data_consistency_error", "采样结果 %s 引用了其他周期", observation.ID)
		}
		if _, ok := siteByID[observation.SiteID]; !ok {
			return result, domain.Conflict("data_consistency_error", "采样结果 %s 引用了不存在的监测点 %s", observation.ID, observation.SiteID)
		}
		if observation.RoundNumber < 1 || observation.RoundNumber > c.PlannedRounds {
			return result, domain.Conflict("data_consistency_error", "采样结果 %s 的计划轮次无效", observation.ID)
		}
		key := fmt.Sprintf("%d|%s", observation.RoundNumber, observation.SiteID)
		if prior, exists := observations[key]; exists {
			return result, domain.Conflict("data_consistency_error", "计划轮次 %d 的监测点 %s 存在重复结果 %s 和 %s", observation.RoundNumber, observation.SiteID, prior.ID, observation.ID)
		}
		observations[key] = observation
	}
	investigationByObservation := map[string]domain.DeviationInvestigation{}
	for _, inv := range c.Investigations {
		if inv.CampaignID != c.ID {
			return result, domain.Conflict("data_consistency_error", "调查 %s 引用了其他周期", inv.ID)
		}
		if prior, exists := investigationByObservation[inv.ObservationID]; exists {
			return result, domain.Conflict("data_consistency_error", "采样结果 %s 重复关联调查 %s 和 %s", inv.ObservationID, prior.ID, inv.ID)
		}
		investigationByObservation[inv.ObservationID] = inv
	}
	recordedTotal := 0
	for round := 1; round <= c.PlannedRounds; round++ {
		summary := domain.SamplingRoundSummary{RoundNumber: round, Required: len(sites)}
		for _, site := range sites {
			key := fmt.Sprintf("%d|%s", round, site.ID)
			observation, present := observations[key]
			cell := domain.SamplingCoverageCell{RoundNumber: round, SiteID: site.ID, AreaName: site.AreaName, PointCode: site.PointCode, Metric: site.Metric, Status: domain.CoverageMissing}
			if present {
				recordedTotal++
				summary.Recorded++
				cell.ObservationID, cell.Explanation = observation.ID, observation.Explanation
				if observation.Verdict == domain.VerdictAlert {
					cell.Status = domain.CoverageAlert
					summary.Alert++
					inv, ok := investigationByObservation[observation.ID]
					if !ok {
						return result, domain.Conflict("data_consistency_error", "超限结果 %s 缺少自动建立的调查", observation.ID)
					}
					if (filter.RoundNumber == 0 || filter.RoundNumber == round) && (filter.AreaName == "" || filter.AreaName == site.AreaName) && (filter.Metric == "" || filter.Metric == site.Metric) {
						result.AlertWorkItems = append(result.AlertWorkItems, domain.AlertWorkItem{ObservationID: observation.ID, SiteID: site.ID, AreaName: site.AreaName, PointCode: site.PointCode, Metric: site.Metric, RoundNumber: round, InvestigationID: inv.ID, InvestigationStatus: inv.Status})
					}
				} else {
					cell.Status = domain.CoverageNormal
					summary.Normal++
				}
			} else {
				summary.Missing++
			}
			if (filter.RoundNumber == 0 || filter.RoundNumber == round) && (filter.AreaName == "" || filter.AreaName == site.AreaName) && (filter.Metric == "" || filter.Metric == site.Metric) {
				result.Cells = append(result.Cells, cell)
			}
		}
		if filter.RoundNumber == 0 || filter.RoundNumber == round {
			result.Rounds = append(result.Rounds, summary)
		}
	}
	result.Evaluable = true
	total := c.PlannedRounds * len(sites)
	if total > 0 {
		result.CompletionRatio = float64(recordedTotal) / float64(total)
	}
	return result, nil
}

func InvestigationPreflight(c *domain.MonitoringCampaign, investigationID string) (domain.InvestigationClosePreflight, error) {
	inv, err := c.Investigation(investigationID)
	if err != nil {
		return domain.InvestigationClosePreflight{}, err
	}
	if inv.CampaignID != c.ID {
		return domain.InvestigationClosePreflight{}, domain.NotFound("偏差调查", investigationID)
	}
	observation, err := c.Observation(inv.ObservationID)
	if err != nil {
		return domain.InvestigationClosePreflight{}, domain.Conflict("data_consistency_error", "调查 %s 引用了不存在的采样结果 %s", inv.ID, inv.ObservationID)
	}
	complete := InvestigationCompleteness(*inv)
	mutableState := c.Status == domain.StatusInvestigation && c.FrozenAt == nil
	return domain.InvestigationClosePreflight{CampaignID: c.ID, Version: c.Version, CampaignStatus: c.Status, Investigation: *inv, Observation: *observation, MissingFields: complete.Missing, CanConclude: mutableState && inv.Status == domain.InvestigationOpen && complete.Complete}, nil
}

func CorrectiveActions(c *domain.MonitoringCampaign, filter domain.CorrectiveActionFilter, now time.Time) (domain.CorrectiveActionList, error) {
	result := domain.CorrectiveActionList{CampaignID: c.ID, Version: c.Version, AsOf: now.UTC(), Items: []domain.CorrectiveActionListItem{}}
	if filter.InvestigationID != "" {
		if _, err := c.Investigation(filter.InvestigationID); err != nil {
			return result, err
		}
	}
	for _, action := range c.Actions {
		completed := action.CompletedAt != nil && strings.TrimSpace(action.CompletionEvidence) != ""
		overdue := !completed && action.DueAt.Before(now)
		if filter.InvestigationID != "" && action.InvestigationID != filter.InvestigationID {
			continue
		}
		if filter.Owner != "" && !strings.EqualFold(action.Owner, filter.Owner) {
			continue
		}
		if filter.Completed != nil && completed != *filter.Completed {
			continue
		}
		if filter.Overdue != nil && overdue != *filter.Overdue {
			continue
		}
		result.Items = append(result.Items, domain.CorrectiveActionListItem{CorrectiveAction: action, Completed: completed, Overdue: overdue})
	}
	sort.Slice(result.Items, func(i, j int) bool {
		if !result.Items[i].DueAt.Equal(result.Items[j].DueAt) {
			return result.Items[i].DueAt.Before(result.Items[j].DueAt)
		}
		return result.Items[i].ID < result.Items[j].ID
	})
	result.Summary.Total = len(result.Items)
	for _, item := range result.Items {
		if item.Completed {
			result.Summary.Completed++
		} else {
			result.Summary.Outstanding++
		}
		if item.Overdue {
			result.Summary.Overdue++
		}
	}
	return result, nil
}

func VerificationPreflight(c *domain.MonitoringCampaign) domain.VerificationStartPreflight {
	result := domain.VerificationStartPreflight{CampaignID: c.ID, Version: c.Version, Status: c.Status, Blockers: []domain.VerificationBlocker{}}
	if c.Status != domain.StatusCorrection {
		result.Blockers = append(result.Blockers, domain.VerificationBlocker{Code: "invalid_campaign_status", Message: fmt.Sprintf("当前状态 %s 不能开始验证", c.Status)})
	}
	for _, inv := range c.Investigations {
		if inv.Status != domain.InvestigationConcluded {
			result.Blockers = append(result.Blockers, domain.VerificationBlocker{Code: "investigation_open", InvestigationID: inv.ID, Message: "调查尚未闭合"})
			continue
		}
		actions := c.ActionsForInvestigation(inv.ID)
		if len(actions) == 0 {
			result.Blockers = append(result.Blockers, domain.VerificationBlocker{Code: "action_missing", InvestigationID: inv.ID, Message: "已闭合调查尚无纠正措施"})
		}
		for _, action := range actions {
			switch {
			case strings.TrimSpace(action.Owner) == "" || strings.TrimSpace(action.Description) == "":
				result.Blockers = append(result.Blockers, domain.VerificationBlocker{Code: "action_material_missing", InvestigationID: inv.ID, ActionID: action.ID, Message: "纠正措施负责人或描述缺失"})
			case action.CompletedAt == nil:
				result.Blockers = append(result.Blockers, domain.VerificationBlocker{Code: "action_incomplete", InvestigationID: inv.ID, ActionID: action.ID, Message: "纠正措施尚未完成"})
			case strings.TrimSpace(action.CompletionEvidence) == "":
				result.Blockers = append(result.Blockers, domain.VerificationBlocker{Code: "completion_evidence_missing", InvestigationID: inv.ID, ActionID: action.ID, Message: "纠正措施缺少完成证据"})
			}
		}
	}
	result.CanStart = len(result.Blockers) == 0
	return result
}

func VerificationRounds(c *domain.MonitoringCampaign, fromRound, toRound int) (domain.VerificationComparison, error) {
	result := domain.VerificationComparison{CampaignID: c.ID, Version: c.Version, Status: c.Status, Sites: []domain.VerificationSiteComparison{}, Rounds: []domain.VerificationRoundSummary{}}
	if c.VerificationRound == 0 {
		if fromRound != 0 || toRound != 0 {
			return result, domain.Validation("invalid_verification_range", "当前尚无验证轮次")
		}
		result.Reason = "尚未开始验证复采"
		return result, nil
	}
	if fromRound == 0 {
		fromRound = 1
	}
	if toRound == 0 {
		toRound = c.VerificationRound
	}
	if fromRound < 1 || toRound < fromRound || toRound > c.VerificationRound {
		return result, domain.Validation("invalid_verification_range", "验证轮次范围必须在 1 到 %d 之间且起止有序", c.VerificationRound)
	}
	sites := orderedSites(c)
	siteByID := map[string]domain.ControlledSite{}
	for _, site := range sites {
		if _, ok := siteByID[site.ID]; ok {
			return result, domain.Conflict("data_consistency_error", "投影中存在重复监测点 %s", site.ID)
		}
		siteByID[site.ID] = site
	}
	byCell := map[string]domain.SampleObservation{}
	invByObservation := map[string]string{}
	for _, inv := range c.Investigations {
		if inv.CampaignID != c.ID {
			return result, domain.Conflict("data_consistency_error", "调查 %s 引用了其他周期", inv.ID)
		}
		if prior, exists := invByObservation[inv.ObservationID]; exists {
			return result, domain.Conflict("data_consistency_error", "验证结果 %s 重复关联调查 %s 和 %s", inv.ObservationID, prior, inv.ID)
		}
		invByObservation[inv.ObservationID] = inv.ID
	}
	for _, observation := range c.Observations {
		if observation.RoundKind != domain.RoundVerification {
			continue
		}
		if observation.CampaignID != c.ID {
			return result, domain.Conflict("data_consistency_error", "验证结果 %s 引用了其他周期", observation.ID)
		}
		if _, ok := siteByID[observation.SiteID]; !ok {
			return result, domain.Conflict("data_consistency_error", "验证结果 %s 引用了不存在的监测点 %s", observation.ID, observation.SiteID)
		}
		if observation.RoundNumber < 1 || observation.RoundNumber > c.VerificationRound {
			return result, domain.Conflict("data_consistency_error", "验证结果 %s 的轮次无效", observation.ID)
		}
		key := fmt.Sprintf("%d|%s", observation.RoundNumber, observation.SiteID)
		if prior, ok := byCell[key]; ok {
			return result, domain.Conflict("data_consistency_error", "验证轮次 %d 的监测点 %s 存在重复结果 %s 和 %s", observation.RoundNumber, observation.SiteID, prior.ID, observation.ID)
		}
		byCell[key] = observation
	}
	for _, site := range sites {
		row := domain.VerificationSiteComparison{SiteID: site.ID, AreaName: site.AreaName, PointCode: site.PointCode, Metric: site.Metric}
		for round := fromRound; round <= toRound; round++ {
			cell := domain.VerificationCell{RoundNumber: round, Missing: true, AlertLimit: site.AlertLimit}
			if observation, ok := byCell[fmt.Sprintf("%d|%s", round, site.ID)]; ok {
				cell.Missing = false
				cell.ObservationID = observation.ID
				cell.ObservedValue = observation.ObservedValue
				cell.Unit = observation.Unit
				cell.Verdict = observation.Verdict
				if observation.Verdict == domain.VerdictAlert {
					invID, ok := invByObservation[observation.ID]
					if !ok {
						return result, domain.Conflict("data_consistency_error", "再次超限结果 %s 缺少调查", observation.ID)
					}
					cell.InvestigationID = invID
				}
			}
			row.Rounds = append(row.Rounds, cell)
		}
		result.Sites = append(result.Sites, row)
	}
	for round := fromRound; round <= toRound; round++ {
		summary := domain.VerificationRoundSummary{RoundNumber: round}
		for _, site := range sites {
			observation, ok := byCell[fmt.Sprintf("%d|%s", round, site.ID)]
			if !ok {
				summary.Missing++
				continue
			}
			if observation.Verdict == domain.VerdictAlert {
				summary.Alert++
				summary.InvestigationIDs = append(summary.InvestigationIDs, invByObservation[observation.ID])
			} else {
				summary.Normal++
			}
		}
		if len(sites) > 0 {
			summary.CoverageRatio = float64(len(sites)-summary.Missing) / float64(len(sites))
		}
		switch {
		case summary.Alert > 0:
			summary.Conclusion = "failed"
		case summary.Missing > 0:
			summary.Conclusion = "incomplete"
		default:
			summary.Conclusion = "passed"
		}
		result.Rounds = append(result.Rounds, summary)
	}
	return result, nil
}

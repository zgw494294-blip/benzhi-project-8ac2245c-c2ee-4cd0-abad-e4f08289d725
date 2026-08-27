package quality

import (
	"sort"

	"subsurface-survey-gate/internal/domain"
)

func CompareScans(base, target domain.QualityScan) domain.ScanDifferenceReport {
	report := domain.ScanDifferenceReport{BaseScanID: base.ID, TargetScanID: target.ID, ByRule: map[string]domain.DifferenceCounts{}, Resolved: []domain.ScanDifferenceItem{}, Persistent: []domain.ScanDifferenceItem{}, New: []domain.ScanDifferenceItem{}}
	baseByKey := make(map[string]domain.ScanFinding, len(base.Findings))
	targetByKey := make(map[string]domain.ScanFinding, len(target.Findings))
	for _, finding := range base.Findings {
		baseByKey[finding.Key] = finding
	}
	for _, finding := range target.Findings {
		targetByKey[finding.Key] = finding
	}
	for key, finding := range baseByKey {
		if current, ok := targetByKey[key]; ok {
			report.Persistent = append(report.Persistent, domain.ScanDifferenceItem{Finding: current})
			continue
		}
		report.Resolved = append(report.Resolved, domain.ScanDifferenceItem{Finding: finding})
	}
	for key, finding := range targetByKey {
		if _, ok := baseByKey[key]; !ok {
			report.New = append(report.New, domain.ScanDifferenceItem{Finding: finding})
		}
	}
	orderItems(report.Resolved)
	orderItems(report.Persistent)
	orderItems(report.New)
	report.Summary = domain.DifferenceCounts{Resolved: len(report.Resolved), Persistent: len(report.Persistent), New: len(report.New)}
	for _, item := range report.Resolved {
		counts := report.ByRule[item.Finding.RuleCode]
		counts.Resolved++
		report.ByRule[item.Finding.RuleCode] = counts
	}
	for _, item := range report.Persistent {
		counts := report.ByRule[item.Finding.RuleCode]
		counts.Persistent++
		report.ByRule[item.Finding.RuleCode] = counts
	}
	for _, item := range report.New {
		counts := report.ByRule[item.Finding.RuleCode]
		counts.New++
		report.ByRule[item.Finding.RuleCode] = counts
	}
	return report
}

func orderItems(items []domain.ScanDifferenceItem) {
	sort.Slice(items, func(i, j int) bool {
		if items[i].Finding.RuleCode != items[j].Finding.RuleCode {
			return items[i].Finding.RuleCode < items[j].Finding.RuleCode
		}
		return items[i].Finding.ObjectRef < items[j].Finding.ObjectRef
	})
}

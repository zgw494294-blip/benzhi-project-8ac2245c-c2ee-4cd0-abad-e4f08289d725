package assessment

import (
	"fmt"

	"cleanroom-release-go/internal/domain"
)

type ObservationDecision struct {
	Verdict     domain.Verdict `json:"verdict"`
	Explanation string         `json:"explanation"`
}

func AssessObservation(site domain.ControlledSite, value float64, unit string) (ObservationDecision, error) {
	if err := domain.ValidateObservationValue(value); err != nil {
		return ObservationDecision{}, err
	}
	if unit != site.Unit {
		return ObservationDecision{}, domain.Validation("unit_mismatch", "采样单位 %s 与监测点单位 %s 不一致", unit, site.Unit)
	}
	if value > site.AlertLimit {
		return ObservationDecision{
			Verdict:     domain.VerdictAlert,
			Explanation: fmt.Sprintf("观测值 %.4g %s 超过告警限值 %.4g %s", value, unit, site.AlertLimit, site.Unit),
		}, nil
	}
	return ObservationDecision{
		Verdict:     domain.VerdictNormal,
		Explanation: fmt.Sprintf("观测值 %.4g %s 未超过告警限值 %.4g %s", value, unit, site.AlertLimit, site.Unit),
	}, nil
}

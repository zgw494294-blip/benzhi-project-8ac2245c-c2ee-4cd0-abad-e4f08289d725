package application

import "subsurface-survey-gate/internal/domain"

func validateMetadata(m Metadata, create bool) error {
	if m.IdempotencyKey == "" {
		return domain.Validation("idempotencyKey", "不能为空")
	}
	if create && m.ExpectedVersion != 0 {
		return domain.Validation("expectedVersion", "创建批次时必须为 0")
	}
	if !create && m.ExpectedVersion < 1 {
		return domain.Validation("expectedVersion", "必须为正整数")
	}
	return nil
}

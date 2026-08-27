package domain

import (
	"encoding/json"
	"fmt"
	"time"
)

type AuditRecord struct {
	Sequence       int64  `json:"sequence"`
	PreviousDigest string `json:"previousDigest"`
	Digest         string `json:"digest"`
	Event          Event  `json:"event"`
}

type AuditEventView struct {
	Sequence       int64         `json:"sequence"`
	EventType      string        `json:"eventType"`
	Actor          string        `json:"actor"`
	OccurredAt     time.Time     `json:"occurredAt"`
	Version        int64         `json:"version"`
	Summary        string        `json:"summary"`
	PreviousDigest string        `json:"previousDigest"`
	Digest         string        `json:"digest"`
	PreviousState  CampaignState `json:"previousState,omitempty"`
	CurrentState   CampaignState `json:"currentState,omitempty"`
}

type ChainValidation struct {
	Valid            bool  `json:"valid"`
	Continuous       bool  `json:"continuous"`
	DigestLinked     bool  `json:"digestLinked"`
	ValidatedFrom    int64 `json:"validatedFrom"`
	ValidatedThrough int64 `json:"validatedThrough"`
}

type AuditTimeline struct {
	CampaignID      string           `json:"campaignId"`
	Version         int64            `json:"version"`
	Events          []AuditEventView `json:"events"`
	NextCursor      string           `json:"nextCursor"`
	ChainValidation ChainValidation  `json:"chainValidation"`
	ChainRoot       string           `json:"chainRoot"`
}

func EventBusinessSummary(event Event) string {
	switch event.Type {
	case "campaign.created":
		return "创建探测批次"
	case "control.registered":
		return "登记控制点"
	case "control.amended":
		return "修订控制点并保留变更事实"
	case "baseline.locked":
		return "锁定控制基准"
	case "observation.registered":
		return "登记管段观测"
	case "observations.batch_registered":
		var facts struct {
			Count int `json:"count"`
		}
		_ = json.Unmarshal(event.Facts, &facts)
		return fmt.Sprintf("批量登记 %d 条管段观测", facts.Count)
	case "quality.scanned":
		return "执行质量扫描并更新问题状态"
	case "rectification.submitted":
		return "提交质量问题整改"
	case "review.submitted":
		return "提交质量复核"
	case "review.decided":
		var facts ReviewDecision
		_ = json.Unmarshal(event.Facts, &facts)
		if facts.Decision == "approve" {
			return "批准质量复核，绑定扫描 " + facts.BoundScanID
		}
		return "退回质量整改，绑定扫描 " + facts.BoundScanID
	case "campaign.frozen":
		return "冻结批准成果快照"
	case "credential.issued":
		return "签发成果准入凭据"
	default:
		return event.Type
	}
}

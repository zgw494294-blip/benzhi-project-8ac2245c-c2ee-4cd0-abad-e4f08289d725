package workflow

import (
	"context"
	"crypto/hmac"
	"errors"
	"fmt"
	"strings"

	"cleanroom-release-go/internal/assessment"
	"cleanroom-release-go/internal/domain"
)

type ReviewCommand struct {
	WriteMeta
	Decision string `json:"decision"`
	Comment  string `json:"comment"`
}

func (s *Service) Review(ctx context.Context, campaignID string, cmd ReviewCommand) (*domain.MonitoringCampaign, error) {
	meta, err := mutation(campaignID, "quality.reviewed", cmd.WriteMeta, cmd)
	if err != nil {
		return nil, err
	}
	return s.repo.Update(ctx, meta, func(c *domain.MonitoringCampaign) error {
		if err := c.EnsureMutable(); err != nil {
			return err
		}
		if err := c.RequireStatus(domain.StatusVerificationPassed); err != nil {
			return err
		}
		if cmd.Decision != "reject" && cmd.Decision != "approve" {
			return domain.Validation("invalid_review_decision", "decision 只能是 reject 或 approve")
		}
		if strings.TrimSpace(cmd.Comment) == "" {
			return domain.Validation("missing_review_comment", "comment 不能为空")
		}
		if strings.TrimSpace(cmd.Actor) == "" {
			return domain.Validation("missing_reviewer", "质量审核员不能为空")
		}
		readiness := assessment.ReviewReadiness(c)
		if !readiness.Complete {
			return domain.Validation("review_not_ready", "审核资料不完整: %s", strings.Join(readiness.Missing, "; "))
		}
		now := s.now().UTC()
		c.Reviews = append(c.Reviews, domain.QualityReview{ID: fmt.Sprintf("review-%d", len(c.Reviews)+1), Reviewer: cmd.Actor, Decision: cmd.Decision, Comment: strings.TrimSpace(cmd.Comment), CreatedAt: now})
		c.AddAudit("quality."+cmd.Decision, cmd.Actor, cmd.Comment, now)
		if cmd.Decision == "reject" {
			return nil
		}
		c.FrozenAt = &now
		c.Status = domain.StatusFrozen
		digest, err := c.ComputeFrozenDigest()
		if err != nil {
			return err
		}
		c.FrozenDigest = digest
		return nil
	})
}

type IssueCredentialCommand struct {
	WriteMeta
	ID       string `json:"id"`
	IssuedBy string `json:"issuedBy"`
}

func (s *Service) IssueCredential(ctx context.Context, campaignID string, cmd IssueCredentialCommand) (domain.ReleaseCredential, error) {
	if strings.TrimSpace(cmd.ID) == "" || strings.TrimSpace(cmd.IssuedBy) == "" {
		return domain.ReleaseCredential{}, domain.Validation("incomplete_credential", "id 和 issuedBy 不能为空")
	}
	if _, err := mutation(campaignID, "credential.issued", cmd.WriteMeta, cmd); err != nil {
		return domain.ReleaseCredential{}, err
	}
	c, err := s.repo.Get(ctx, campaignID)
	if err != nil {
		return domain.ReleaseCredential{}, err
	}
	if c.Status != domain.StatusFrozen || c.FrozenAt == nil || c.FrozenDigest == "" {
		return domain.ReleaseCredential{}, domain.InvalidState("仅已冻结且验证合格的周期可以签发凭据")
	}
	if cmd.ExpectedVersion != c.Version {
		return domain.ReleaseCredential{}, domain.Conflict("version_conflict", "expectedVersion=%d，当前版本=%d", cmd.ExpectedVersion, c.Version)
	}
	if existing, getErr := s.repo.GetCredential(ctx, cmd.ID); getErr == nil {
		if existing.CampaignID == campaignID && existing.IssuedBy == strings.TrimSpace(cmd.IssuedBy) {
			return existing, nil
		}
		return domain.ReleaseCredential{}, domain.Conflict("credential_exists", "凭据 %s 已存在且内容不同", cmd.ID)
	} else {
		var de *domain.Error
		if !errors.As(getErr, &de) || de.Kind != domain.KindNotFound {
			return domain.ReleaseCredential{}, getErr
		}
	}
	digest, err := c.ComputeFrozenDigest()
	if err != nil {
		return domain.ReleaseCredential{}, err
	}
	if digest != c.FrozenDigest {
		return domain.ReleaseCredential{}, domain.Conflict("snapshot_digest_mismatch", "冻结快照摘要不一致")
	}
	credential := domain.ReleaseCredential{ID: cmd.ID, CampaignID: c.ID, CampaignVersion: c.Version, SnapshotDigest: c.FrozenDigest, IssuedBy: strings.TrimSpace(cmd.IssuedBy), IssuedAt: s.now().UTC()}
	credential.Signature = s.signature(credential)
	if err := s.repo.SaveCredential(ctx, credential); err != nil {
		return domain.ReleaseCredential{}, err
	}
	return credential, nil
}

func (s *Service) VerifyCredential(ctx context.Context, id string) (domain.CredentialVerification, error) {
	credential, err := s.repo.GetCredential(ctx, id)
	if err != nil {
		return domain.CredentialVerification{}, err
	}
	result := domain.CredentialVerification{CredentialID: id, CampaignID: credential.CampaignID}
	c, err := s.repo.Get(ctx, credential.CampaignID)
	if err != nil {
		result.Reason = "凭据引用的监测周期不存在"
		return result, nil
	}
	digest, err := c.ComputeFrozenDigest()
	if err != nil {
		return result, err
	}
	expected := s.signature(credential)
	if c.Status != domain.StatusFrozen || credential.CampaignVersion != c.Version || credential.SnapshotDigest != c.FrozenDigest || digest != c.FrozenDigest || !hmac.Equal([]byte(expected), []byte(credential.Signature)) {
		result.Reason = "凭据、签名或冻结快照不一致"
		return result, nil
	}
	result.Valid, result.Reason = true, "凭据有效，且与冻结快照一致"
	return result, nil
}

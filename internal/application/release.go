package application

import (
	"context"
	"errors"
	"net/http"

	"subsurface-survey-gate/internal/domain"
)

func (s *Service) Freeze(ctx context.Context, campaignID string, cmd FreezeCampaign) (Result, error) {
	if err := validateMetadata(cmd.Metadata, false); err != nil {
		return Result{}, err
	}
	fp, replay, err := s.prepare(ctx, campaignID, cmd.IdempotencyKey, cmd)
	if err != nil {
		return Result{}, err
	}
	if replay != nil {
		return *replay, nil
	}
	c, err := s.store.Load(ctx, campaignID)
	if err != nil {
		return Result{}, err
	}
	root, err := s.store.ChainRoot(ctx, campaignID)
	if err != nil {
		return Result{}, err
	}
	digest := domain.SnapshotDigest(c)
	now := s.now().UTC()
	previousState := c.State
	if err := c.Freeze(digest, root, now); err != nil {
		return Result{}, err
	}
	e := domain.NewStateEvent(c, newID("evt"), "campaign.frozen", cmd.Actor, now, previousState, c.Frozen)
	return s.commit(ctx, campaignID, cmd.ExpectedVersion, c, []domain.Event{e}, cmd.IdempotencyKey, fp, http.StatusOK, c)
}

func (s *Service) IssueCredential(ctx context.Context, campaignID string, cmd IssueCredential) (Result, error) {
	if err := validateMetadata(cmd.Metadata, false); err != nil {
		return Result{}, err
	}
	fp, replay, err := s.prepare(ctx, campaignID, cmd.IdempotencyKey, cmd)
	if err != nil {
		return Result{}, err
	}
	if replay != nil {
		return *replay, nil
	}
	if cmd.IssuedBy == "" {
		return Result{}, domain.Validation("issuedBy", "不能为空")
	}
	c, err := s.store.Load(ctx, campaignID)
	if err != nil {
		return Result{}, err
	}
	if c.Frozen == nil {
		return Result{}, domain.Conflict("批次尚未冻结")
	}
	now := s.now().UTC()
	previousState := c.State
	credential := domain.ReleaseCredential{ID: newID("cred"), CampaignID: c.ID, FrozenVersion: c.Frozen.FrozenVersion, SnapshotDigest: c.Frozen.SnapshotDigest, EventChainRoot: c.Frozen.EventChainRoot, IssuedBy: cmd.IssuedBy, IssuedAt: now}
	credential.VerificationCode = domain.CredentialCode(credential)
	if err := c.IssueCredential(credential, now); err != nil {
		return Result{}, err
	}
	e := domain.NewStateEvent(c, newID("evt"), "credential.issued", cmd.IssuedBy, now, previousState, credential)
	return s.commit(ctx, campaignID, cmd.ExpectedVersion, c, []domain.Event{e}, cmd.IdempotencyKey, fp, http.StatusCreated, credential)
}

func (s *Service) VerifyCredential(ctx context.Context, supplied domain.ReleaseCredential) (domain.CredentialVerification, error) {
	result := domain.CredentialVerification{CredentialID: supplied.ID, CampaignID: supplied.CampaignID, FrozenVersion: supplied.FrozenVersion}
	if supplied.ID == "" || supplied.VerificationCode == "" || domain.CredentialCode(supplied) != supplied.VerificationCode {
		result.Reason = "凭据字段或校验码无效"
		return result, nil
	}
	stored, err := s.store.FindCredential(ctx, supplied.ID)
	if err != nil {
		var de *domain.Error
		if errors.As(err, &de) && de.Kind == domain.ErrorNotFound {
			result.Reason = "凭据未在本地账本登记"
			return result, nil
		}
		return domain.CredentialVerification{}, err
	}
	if domain.Digest(stored) != domain.Digest(&supplied) {
		result.Reason = "凭据与已签发记录不一致"
		return result, nil
	}
	campaign, err := s.store.Load(ctx, supplied.CampaignID)
	if err != nil {
		var de *domain.Error
		if errors.As(err, &de) && de.Kind == domain.ErrorNotFound {
			result.Reason = "冻结快照摘要或事件链根不一致"
			return result, nil
		}
		return domain.CredentialVerification{}, err
	}
	if campaign.Frozen == nil || campaign.Frozen.FrozenVersion != supplied.FrozenVersion || campaign.Frozen.EventChainRoot != supplied.EventChainRoot || campaign.Frozen.SnapshotDigest != supplied.SnapshotDigest || domain.SnapshotDigest(campaign) != supplied.SnapshotDigest {
		result.Reason = "冻结快照摘要或事件链根不一致"
		return result, nil
	}
	result.Valid, result.Reason = true, "凭据有效"
	return result, nil
}

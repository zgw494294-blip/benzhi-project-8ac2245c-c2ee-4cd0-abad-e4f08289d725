package application

import (
	"context"
	"net/http"

	"subsurface-survey-gate/internal/domain"
)

func (s *Service) SubmitReview(ctx context.Context, campaignID string, cmd SubmitReview) (Result, error) {
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
	now := s.now().UTC()
	previousState := c.State
	if err := c.SubmitReview(cmd.Submitter, now); err != nil {
		return Result{}, err
	}
	e := domain.NewStateEvent(c, newID("evt"), "review.submitted", cmd.Submitter, now, previousState, map[string]any{"scanId": c.LatestScan().ID})
	return s.commit(ctx, campaignID, cmd.ExpectedVersion, c, []domain.Event{e}, cmd.IdempotencyKey, fp, http.StatusOK, c)
}

func (s *Service) DecideReview(ctx context.Context, campaignID string, cmd DecideReview) (Result, error) {
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
	now := s.now().UTC()
	previousState := c.State
	id := newID("rev")
	if err := c.DecideReview(id, cmd.Reviewer, cmd.Decision, cmd.Reason, now); err != nil {
		return Result{}, err
	}
	d := c.Reviews[len(c.Reviews)-1]
	e := domain.NewStateEvent(c, newID("evt"), "review.decided", cmd.Reviewer, now, previousState, d)
	return s.commit(ctx, campaignID, cmd.ExpectedVersion, c, []domain.Event{e}, cmd.IdempotencyKey, fp, http.StatusOK, c)
}

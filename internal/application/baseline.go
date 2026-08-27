package application

import (
	"context"
	"net/http"

	"subsurface-survey-gate/internal/domain"
)

func (s *Service) AddControl(ctx context.Context, campaignID string, cmd AddControl) (Result, error) {
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
	p := domain.ControlPoint{ID: newID("ctl"), CampaignID: campaignID, Code: cmd.Code, Easting: cmd.Easting, Northing: cmd.Northing, Elevation: cmd.Elevation, Source: cmd.Source, VerifiedBy: cmd.VerifiedBy, VerifiedAt: cmd.VerifiedAt.UTC()}
	if err := c.AddControl(p, now); err != nil {
		return Result{}, err
	}
	e := domain.NewEvent(c, newID("evt"), "control.registered", cmd.Actor, now, p)
	return s.commit(ctx, campaignID, cmd.ExpectedVersion, c, []domain.Event{e}, cmd.IdempotencyKey, fp, http.StatusCreated, c)
}

func (s *Service) LockBaseline(ctx context.Context, campaignID string, cmd LockBaseline) (Result, error) {
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
	readiness := s.scanner.BaselineReadiness(c)
	if !readiness.ReadyToLock {
		return Result{}, domain.Conflict("控制基准预检存在阻断项，当前不可锁定")
	}
	now := s.now().UTC()
	previousState := c.State
	if err := c.LockBaseline(now); err != nil {
		return Result{}, err
	}
	e := domain.NewStateEvent(c, newID("evt"), "baseline.locked", cmd.Actor, now, previousState, map[string]any{"controlCount": len(c.Controls)})
	return s.commit(ctx, campaignID, cmd.ExpectedVersion, c, []domain.Event{e}, cmd.IdempotencyKey, fp, http.StatusOK, c)
}

func (s *Service) BaselineReadiness(ctx context.Context, campaignID string) (domain.BaselineReadiness, error) {
	c, err := s.store.Load(ctx, campaignID)
	if err != nil {
		return domain.BaselineReadiness{}, err
	}
	if c.State != domain.StateDraft {
		return domain.BaselineReadiness{}, domain.Conflict("只能对草稿批次执行控制基准锁定预检")
	}
	return s.scanner.BaselineReadiness(c), nil
}

func (s *Service) AmendControl(ctx context.Context, campaignID, controlID string, cmd AmendControl) (Result, error) {
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
	replacement := domain.ControlPoint{ID: controlID, CampaignID: campaignID, Code: cmd.Code, Easting: cmd.Easting, Northing: cmd.Northing, Elevation: cmd.Elevation, Source: cmd.Source, VerifiedBy: cmd.VerifiedBy, VerifiedAt: cmd.VerifiedAt.UTC()}
	if err := c.AmendControl(newID("ctlchg"), controlID, cmd.Reason, cmd.Actor, replacement, now); err != nil {
		return Result{}, err
	}
	change := c.ControlChanges[len(c.ControlChanges)-1]
	e := domain.NewEvent(c, newID("evt"), "control.amended", cmd.Actor, now, change)
	if previousState != c.State {
		e.PreviousState, e.CurrentState = previousState, c.State
	}
	return s.commit(ctx, campaignID, cmd.ExpectedVersion, c, []domain.Event{e}, cmd.IdempotencyKey, fp, http.StatusOK, c)
}

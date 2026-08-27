package application

import (
	"context"
	"net/http"

	"subsurface-survey-gate/internal/domain"
)

func (s *Service) AddObservation(ctx context.Context, campaignID string, cmd AddObservation) (Result, error) {
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
	o := domain.PipelineObservation{ID: newID("obs"), CampaignID: campaignID, SegmentCode: cmd.SegmentCode, UtilityType: cmd.UtilityType, StartPointID: cmd.StartPointID, EndPointID: cmd.EndPointID, BurialDepthMM: cmd.BurialDepthMM, DiameterMM: cmd.DiameterMM, Material: cmd.Material, DetectionMethod: cmd.DetectionMethod, ObservedAt: cmd.ObservedAt.UTC()}
	if err := c.AddObservation(o, now); err != nil {
		return Result{}, err
	}
	o = c.Observations[len(c.Observations)-1]
	e := domain.NewEvent(c, newID("evt"), "observation.registered", cmd.Actor, now, o)
	return s.commit(ctx, campaignID, cmd.ExpectedVersion, c, []domain.Event{e}, cmd.IdempotencyKey, fp, http.StatusCreated, c)
}

type ObservationBatchResult struct {
	CampaignID   string                       `json:"campaignId"`
	Version      int64                        `json:"version"`
	Observations []domain.PipelineObservation `json:"observations"`
}

func (s *Service) AddObservationBatch(ctx context.Context, campaignID string, cmd AddObservationBatch) (Result, error) {
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
	if len(cmd.Observations) < 1 || len(cmd.Observations) > 100 {
		return Result{}, domain.Validation("observations", "数量必须为 1 到 100 条")
	}
	c, err := s.store.Load(ctx, campaignID)
	if err != nil {
		return Result{}, err
	}
	now := s.now().UTC()
	candidates := make([]domain.PipelineObservation, len(cmd.Observations))
	for i, input := range cmd.Observations {
		candidates[i] = domain.PipelineObservation{ID: newID("obs"), CampaignID: campaignID, SegmentCode: input.SegmentCode, UtilityType: input.UtilityType, StartPointID: input.StartPointID, EndPointID: input.EndPointID, BurialDepthMM: input.BurialDepthMM, DiameterMM: input.DiameterMM, Material: input.Material, DetectionMethod: input.DetectionMethod, ObservedAt: input.ObservedAt.UTC()}
	}
	if err := c.AddObservations(candidates, now); err != nil {
		return Result{}, err
	}
	added := append([]domain.PipelineObservation(nil), c.Observations[len(c.Observations)-len(candidates):]...)
	codes := make([]string, len(added))
	for i := range added {
		codes[i] = added[i].SegmentCode
	}
	body := ObservationBatchResult{CampaignID: c.ID, Version: c.Version, Observations: added}
	e := domain.NewEvent(c, newID("evt"), "observations.batch_registered", cmd.Actor, now, map[string]any{"count": len(added), "segmentCodes": codes})
	return s.commit(ctx, campaignID, cmd.ExpectedVersion, c, []domain.Event{e}, cmd.IdempotencyKey, fp, http.StatusCreated, body)
}

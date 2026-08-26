package application

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"subsurface-survey-gate/internal/domain"
)

type Service struct {
	store   Store
	scanner QualityScanner
	now     func() time.Time
}

func NewService(store Store, scanner QualityScanner) *Service {
	return &Service{store: store, scanner: scanner, now: time.Now}
}

func fingerprint(v any) string { return domain.Digest(v) }

func response(status int, value any) Result {
	b, _ := json.Marshal(value)
	return Result{Status: status, Body: b}
}

func (s *Service) replay(ctx context.Context, namespace, key, fp string) (*Result, error) {
	record, err := s.store.LookupIdempotency(ctx, namespace, key)
	if err != nil {
		return nil, err
	}
	if record == nil {
		return nil, nil
	}
	if record.Fingerprint != fp {
		return nil, domain.IdempotencyConflict()
	}
	return &Result{Status: record.Status, Body: record.Response, Replayed: true}, nil
}

func (s *Service) prepare(ctx context.Context, campaignID, key string, command any) (string, *Result, error) {
	fp := fingerprint(command)
	replay, err := s.replay(ctx, campaignID, key, fp)
	return fp, replay, err
}

func (s *Service) commit(ctx context.Context, id string, expected int64, c *domain.SurveyCampaign, events []domain.Event, key, fp string, status int, body any) (Result, error) {
	b, _ := json.Marshal(body)
	namespace := id
	if expected == 0 {
		namespace = "__create__"
	}
	record := domain.IdempotencyRecord{CampaignID: namespace, Key: key, Fingerprint: fp, Status: status, Response: b}
	_, replay, err := s.store.Commit(ctx, id, expected, c, events, record)
	if err != nil {
		return Result{}, err
	}
	if replay != nil {
		if replay.Fingerprint != fp {
			return Result{}, domain.IdempotencyConflict()
		}
		return Result{Status: replay.Status, Body: replay.Response, Replayed: true}, nil
	}
	return Result{Status: status, Body: b}, nil
}

func (s *Service) CreateCampaign(ctx context.Context, cmd CreateCampaign) (Result, error) {
	if err := validateMetadata(cmd.Metadata, true); err != nil {
		return Result{}, err
	}
	fp := fingerprint(cmd)
	if replay, err := s.replay(ctx, "__create__", cmd.IdempotencyKey, fp); err != nil {
		return Result{}, err
	} else if replay != nil {
		return *replay, nil
	}
	now, id := s.now().UTC(), newID("cmp")
	c, err := domain.NewCampaign(id, cmd.Name, cmd.SurveyArea, cmd.CoordinateReference, cmd.SpecificationRevision, now)
	if err != nil {
		return Result{}, err
	}
	e := domain.NewStateEvent(c, newID("evt"), "campaign.created", cmd.Actor, now, "", c)
	return s.commit(ctx, id, 0, c, []domain.Event{e}, cmd.IdempotencyKey, fp, http.StatusCreated, c)
}

func (s *Service) Campaign(ctx context.Context, id string) (*domain.SurveyCampaign, error) {
	return s.store.Load(ctx, id)
}

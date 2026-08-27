package domain

import (
	"encoding/json"
	"time"
)

type Event struct {
	ID            string          `json:"id"`
	CampaignID    string          `json:"campaignId"`
	Type          string          `json:"type"`
	Version       int64           `json:"version"`
	Actor         string          `json:"actor,omitempty"`
	OccurredAt    time.Time       `json:"occurredAt"`
	Facts         json.RawMessage `json:"facts,omitempty"`
	PreviousState CampaignState   `json:"previousState,omitempty"`
	CurrentState  CampaignState   `json:"currentState,omitempty"`
}

func NewStateEvent(c *SurveyCampaign, id, eventType, actor string, at time.Time, previous CampaignState, facts any) Event {
	e := NewEvent(c, id, eventType, actor, at, facts)
	e.PreviousState, e.CurrentState = previous, c.State
	return e
}

func NewEvent(c *SurveyCampaign, id, eventType, actor string, at time.Time, facts any) Event {
	b, _ := json.Marshal(facts)
	return Event{ID: id, CampaignID: c.ID, Type: eventType, Version: c.Version, Actor: actor, OccurredAt: at.UTC(), Facts: b}
}

type IdempotencyRecord struct {
	CampaignID  string          `json:"campaignId"`
	Key         string          `json:"key"`
	Fingerprint string          `json:"fingerprint"`
	Status      int             `json:"status"`
	Response    json.RawMessage `json:"response"`
}

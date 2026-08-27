package domain

import (
	"math"
	"sort"
	"time"
)

type CoordinateRange struct {
	MinEasting  float64 `json:"minEasting"`
	MaxEasting  float64 `json:"maxEasting"`
	MinNorthing float64 `json:"minNorthing"`
	MaxNorthing float64 `json:"maxNorthing"`
}

type ControlReadinessFact struct {
	ID         string    `json:"id"`
	Code       string    `json:"code"`
	Easting    float64   `json:"easting"`
	Northing   float64   `json:"northing"`
	Elevation  float64   `json:"elevation"`
	Verified   bool      `json:"verified"`
	VerifiedBy string    `json:"verifiedBy"`
	VerifiedAt time.Time `json:"verifiedAt"`
}

type BaselineReadinessSummary struct {
	ControlCount        int                    `json:"controlCount"`
	Controls            []ControlReadinessFact `json:"controls"`
	CoordinateRange     *CoordinateRange       `json:"coordinateRange"`
	MinimumPointSpacing float64                `json:"minimumPointSpacing"`
}

type ReadinessIssue struct {
	RuleCode     string        `json:"ruleCode"`
	Severity     IssueSeverity `json:"severity"`
	ObjectRef    string        `json:"objectRef"`
	ControlCodes []string      `json:"controlCodes"`
	Field        string        `json:"field,omitempty"`
	Description  string        `json:"description"`
}

type BaselineReadiness struct {
	CampaignID      string                   `json:"campaignId"`
	Version         int64                    `json:"version"`
	ReadyToLock     bool                     `json:"readyToLock"`
	ReadinessDigest string                   `json:"readinessDigest"`
	Summary         BaselineReadinessSummary `json:"summary"`
	Issues          []ReadinessIssue         `json:"issues"`
}

func NormalizedControlFacts(c *SurveyCampaign) []ControlReadinessFact {
	facts := make([]ControlReadinessFact, 0, len(c.Controls))
	for _, p := range c.Controls {
		verified := p.VerifiedBy != "" && !p.VerifiedAt.IsZero() && !p.VerifiedAt.After(c.UpdatedAt.Add(5*time.Minute))
		facts = append(facts, ControlReadinessFact{ID: p.ID, Code: p.Code, Easting: p.Easting, Northing: p.Northing, Elevation: p.Elevation, Verified: verified, VerifiedBy: p.VerifiedBy, VerifiedAt: p.VerifiedAt.UTC()})
	}
	sort.Slice(facts, func(i, j int) bool {
		if facts[i].Code != facts[j].Code {
			return facts[i].Code < facts[j].Code
		}
		return facts[i].ID < facts[j].ID
	})
	return facts
}

func ControlReadinessSummary(c *SurveyCampaign) BaselineReadinessSummary {
	facts := NormalizedControlFacts(c)
	summary := BaselineReadinessSummary{ControlCount: len(facts), Controls: facts}
	if len(facts) == 0 {
		return summary
	}
	r := &CoordinateRange{MinEasting: facts[0].Easting, MaxEasting: facts[0].Easting, MinNorthing: facts[0].Northing, MaxNorthing: facts[0].Northing}
	minimum := 0.0
	if len(facts) > 1 {
		minimum = math.Inf(1)
	}
	for i, p := range facts {
		r.MinEasting = math.Min(r.MinEasting, p.Easting)
		r.MaxEasting = math.Max(r.MaxEasting, p.Easting)
		r.MinNorthing = math.Min(r.MinNorthing, p.Northing)
		r.MaxNorthing = math.Max(r.MaxNorthing, p.Northing)
		for j := i + 1; j < len(facts); j++ {
			minimum = math.Min(minimum, math.Hypot(p.Easting-facts[j].Easting, p.Northing-facts[j].Northing))
		}
	}
	summary.CoordinateRange, summary.MinimumPointSpacing = r, minimum
	return summary
}

func BaselineReadinessDigest(c *SurveyCampaign) string {
	return Digest(struct {
		CampaignID string                 `json:"campaignId"`
		Version    int64                  `json:"version"`
		Controls   []ControlReadinessFact `json:"controls"`
	}{c.ID, c.Version, NormalizedControlFacts(c)})
}

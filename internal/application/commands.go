package application

import (
	"encoding/json"
	"time"

	"subsurface-survey-gate/internal/domain"
)

type Metadata struct {
	ExpectedVersion int64  `json:"expectedVersion"`
	IdempotencyKey  string `json:"idempotencyKey"`
}

type Result struct {
	Status   int
	Body     json.RawMessage
	Replayed bool
}

type CreateCampaign struct {
	Metadata
	Name                  string `json:"name"`
	SurveyArea            string `json:"surveyArea"`
	CoordinateReference   string `json:"coordinateReference"`
	SpecificationRevision string `json:"specificationRevision"`
	Actor                 string `json:"actor"`
}
type AddControl struct {
	Metadata
	Code       string    `json:"code"`
	Easting    float64   `json:"easting"`
	Northing   float64   `json:"northing"`
	Elevation  float64   `json:"elevation"`
	Source     string    `json:"source"`
	VerifiedBy string    `json:"verifiedBy"`
	VerifiedAt time.Time `json:"verifiedAt"`
	Actor      string    `json:"actor"`
}
type AmendControl struct {
	Metadata
	Code       string    `json:"code"`
	Easting    float64   `json:"easting"`
	Northing   float64   `json:"northing"`
	Elevation  float64   `json:"elevation"`
	Source     string    `json:"source"`
	VerifiedBy string    `json:"verifiedBy"`
	VerifiedAt time.Time `json:"verifiedAt"`
	Reason     string    `json:"reason"`
	Actor      string    `json:"actor"`
}
type LockBaseline struct {
	Metadata
	Actor string `json:"actor"`
}
type AddObservation struct {
	Metadata
	SegmentCode     string    `json:"segmentCode"`
	UtilityType     string    `json:"utilityType"`
	StartPointID    string    `json:"startPointId"`
	EndPointID      string    `json:"endPointId"`
	BurialDepthMM   int       `json:"burialDepthMm"`
	DiameterMM      int       `json:"diameterMm"`
	Material        string    `json:"material"`
	DetectionMethod string    `json:"detectionMethod"`
	ObservedAt      time.Time `json:"observedAt"`
	Actor           string    `json:"actor"`
}
type ObservationInput struct {
	SegmentCode     string    `json:"segmentCode"`
	UtilityType     string    `json:"utilityType"`
	StartPointID    string    `json:"startPointId"`
	EndPointID      string    `json:"endPointId"`
	BurialDepthMM   int       `json:"burialDepthMm"`
	DiameterMM      int       `json:"diameterMm"`
	Material        string    `json:"material"`
	DetectionMethod string    `json:"detectionMethod"`
	ObservedAt      time.Time `json:"observedAt"`
}
type AddObservationBatch struct {
	Metadata
	Observations []ObservationInput `json:"observations"`
	Actor        string             `json:"actor"`
}
type RunScan struct {
	Metadata
	Actor string `json:"actor"`
}
type SubmitRectification struct {
	Metadata
	IssueID            string                      `json:"issueId"`
	Note               string                      `json:"note"`
	Actor              string                      `json:"actor"`
	RevisedObservation *domain.PipelineObservation `json:"revisedObservation,omitempty"`
}
type SubmitReview struct {
	Metadata
	Submitter string `json:"submitter"`
}
type DecideReview struct {
	Metadata
	Reviewer string `json:"reviewer"`
	Decision string `json:"decision"`
	Reason   string `json:"reason"`
}
type FreezeCampaign struct {
	Metadata
	Actor string `json:"actor"`
}
type IssueCredential struct {
	Metadata
	IssuedBy string `json:"issuedBy"`
}
type VerifyCredential struct {
	Credential domain.ReleaseCredential `json:"credential"`
}

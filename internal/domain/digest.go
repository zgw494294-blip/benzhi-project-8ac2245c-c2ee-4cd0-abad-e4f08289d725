package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
)

func Digest(v any) string {
	b, _ := json.Marshal(v)
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

// SnapshotDigest 只覆盖批准时的业务事实，排除随后产生的冻结、凭据和传输状态字段。
func SnapshotDigest(c *SurveyCampaign) string {
	return Digest(struct {
		ID                    string                `json:"id"`
		Name                  string                `json:"name"`
		SurveyArea            string                `json:"surveyArea"`
		CoordinateReference   string                `json:"coordinateReference"`
		SpecificationRevision string                `json:"specificationRevision"`
		Controls              []ControlPoint        `json:"controls"`
		ControlChanges        []ControlChange       `json:"controlChanges"`
		Observations          []PipelineObservation `json:"observations"`
		Issues                []QualityIssue        `json:"issues"`
		Scans                 []QualityScan         `json:"scans"`
		Rectifications        []Rectification       `json:"rectifications"`
		Reviews               []ReviewDecision      `json:"reviews"`
	}{c.ID, c.Name, c.SurveyArea, c.CoordinateReference, c.SpecificationRevision, c.Controls, c.ControlChanges, c.Observations, c.Issues, c.Scans, c.Rectifications, c.Reviews})
}

func CredentialCode(c ReleaseCredential) string {
	return Digest(struct {
		ID             string `json:"id"`
		CampaignID     string `json:"campaignId"`
		FrozenVersion  int64  `json:"frozenVersion"`
		SnapshotDigest string `json:"snapshotDigest"`
		EventChainRoot string `json:"eventChainRoot"`
		IssuedBy       string `json:"issuedBy"`
		IssuedAt       string `json:"issuedAt"`
	}{c.ID, c.CampaignID, c.FrozenVersion, c.SnapshotDigest, c.EventChainRoot, c.IssuedBy, c.IssuedAt.UTC().Format("2006-01-02T15:04:05.000000000Z")})
}

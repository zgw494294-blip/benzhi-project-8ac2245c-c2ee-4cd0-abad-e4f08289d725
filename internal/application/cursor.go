package application

import (
	"encoding/base64"
	"encoding/json"

	"subsurface-survey-gate/internal/domain"
)

type pageCursor struct {
	Kind              string `json:"kind"`
	CampaignID        string `json:"campaignId"`
	Version           int64  `json:"version,omitempty"`
	FilterFingerprint string `json:"filterFingerprint"`
	Offset            int    `json:"offset"`
	Checksum          string `json:"checksum"`
}

func encodeCursor(cursor pageCursor) string {
	cursor.Checksum = ""
	cursor.Checksum = domain.Digest(cursor)
	b, _ := json.Marshal(cursor)
	return base64.RawURLEncoding.EncodeToString(b)
}

func decodeCursor(value string) (pageCursor, error) {
	var cursor pageCursor
	b, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil || json.Unmarshal(b, &cursor) != nil || cursor.Offset < 0 || cursor.Checksum == "" {
		return pageCursor{}, domain.QueryError("cursor", "游标格式无效")
	}
	checksum := cursor.Checksum
	cursor.Checksum = ""
	if domain.Digest(cursor) != checksum {
		return pageCursor{}, domain.QueryError("cursor", "游标校验失败")
	}
	cursor.Checksum = checksum
	return cursor, nil
}

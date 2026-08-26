package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type selfcheckClient struct {
	base    string
	client  *http.Client
	version int64
	key     int
}
type campaignEnvelope struct {
	Campaign struct {
		ID      string `json:"id"`
		Status  string `json:"status"`
		Version int64  `json:"version"`
	} `json:"campaign"`
	Version int64 `json:"version"`
}

func selfcheck(ctx context.Context, base string) error {
	c := &selfcheckClient{base: base, client: &http.Client{Timeout: 3 * time.Second}}
	if err := c.waitReady(ctx); err != nil {
		return err
	}
	due := time.Now().UTC().Add(24 * time.Hour).Format(time.RFC3339)
	steps := []struct {
		method string
		path   string
		body   map[string]any
		want   string
	}{
		{http.MethodPost, "/api/v1/campaigns", map[string]any{"id": "selfcheck-campaign", "facilityName": "自检洁净厂房", "sites": []map[string]any{{"id": "site-a", "areaName": "灌装间", "pointCode": "P-01", "cleanlinessGrade": "B", "metric": "airborne_microbe", "unit": "cfu/m3", "alertLimit": 10}}}, "draft"},
		{http.MethodPost, "/api/v1/campaigns/selfcheck-campaign/plan/lock", map[string]any{"reviewer": "方案复核员", "plannedRounds": 1}, "plan_locked"},
		{http.MethodPost, "/api/v1/campaigns/selfcheck-campaign/observations/planned", map[string]any{"id": "sample-alert", "siteId": "site-a", "roundNumber": 1, "observedValue": 12, "unit": "cfu/m3"}, "investigation"},
		{http.MethodPatch, "/api/v1/campaigns/selfcheck-campaign/investigations/inv-sample-alert/draft", map[string]any{"impactScope": " 灌装间当班产品 ", "hypotheses": []string{"过滤器密封失效", "过滤器密封失效"}, "evidenceRefs": []string{"evidence://pressure-01"}}, "investigation"},
		{http.MethodPost, "/api/v1/campaigns/selfcheck-campaign/investigations/inv-sample-alert/conclude", map[string]any{"rootCause": "过滤器密封件老化"}, "correction"},
		{http.MethodPost, "/api/v1/campaigns/selfcheck-campaign/corrective-actions/batch", map[string]any{"actions": []map[string]any{{"id": "action-1", "investigationId": "inv-sample-alert", "description": "更换密封件并检漏", "owner": "设备负责人", "dueAt": due}}}, "correction"},
		{http.MethodPost, "/api/v1/campaigns/selfcheck-campaign/corrective-actions/batch-complete", map[string]any{"actions": []map[string]any{{"id": "action-1", "completionEvidence": "evidence://leak-test-01"}}}, "correction"},
		{http.MethodPost, "/api/v1/campaigns/selfcheck-campaign/verifications", map[string]any{}, "verification"},
		{http.MethodPost, "/api/v1/campaigns/selfcheck-campaign/observations/verification", map[string]any{"id": "verify-alert", "siteId": "site-a", "roundNumber": 1, "observedValue": 11, "unit": "cfu/m3"}, "investigation"},
		{http.MethodPatch, "/api/v1/campaigns/selfcheck-campaign/investigations/inv-verify-alert/draft", map[string]any{"impactScope": "灌装间复采区域", "hypotheses": []string{"消毒接触时间不足"}, "evidenceRefs": []string{"evidence://disinfection-01"}}, "investigation"},
		{http.MethodPost, "/api/v1/campaigns/selfcheck-campaign/investigations/inv-verify-alert/conclude", map[string]any{"rootCause": "消毒接触时间未达要求"}, "correction"},
		{http.MethodPost, "/api/v1/campaigns/selfcheck-campaign/corrective-actions", map[string]any{"id": "action-2", "investigationId": "inv-verify-alert", "description": "修订消毒计时并再培训", "owner": "环境监测主管", "dueAt": due}, "correction"},
		{http.MethodPost, "/api/v1/campaigns/selfcheck-campaign/corrective-actions/action-2/complete", map[string]any{"completionEvidence": "evidence://training-02"}, "correction"},
		{http.MethodPost, "/api/v1/campaigns/selfcheck-campaign/verifications", map[string]any{}, "verification"},
		{http.MethodPost, "/api/v1/campaigns/selfcheck-campaign/observations/verification", map[string]any{"id": "verify-pass", "siteId": "site-a", "roundNumber": 2, "observedValue": 2, "unit": "cfu/m3"}, "verification_passed"},
		{http.MethodPost, "/api/v1/campaigns/selfcheck-campaign/reviews", map[string]any{"decision": "reject", "comment": "请补充复核说明；自检模拟退回"}, "verification_passed"},
		{http.MethodPost, "/api/v1/campaigns/selfcheck-campaign/reviews", map[string]any{"decision": "approve", "comment": "资料完整，同意复产"}, "frozen"},
	}
	for i, step := range steps {
		c.key = i + 1
		if err := c.writeCampaign(ctx, step.method, step.path, step.body, step.want); err != nil {
			return fmt.Errorf("selfcheck 步骤 %d: %w", i+1, err)
		}
	}
	c.key++
	credentialBody := map[string]any{"id": "credential-selfcheck", "issuedBy": "质量放行审核员"}
	c.addMeta(credentialBody)
	var issued struct {
		Credential struct {
			ID string `json:"id"`
		} `json:"credential"`
	}
	if err := c.request(ctx, http.MethodPost, "/api/v1/campaigns/selfcheck-campaign/credentials", credentialBody, http.StatusCreated, &issued); err != nil {
		return err
	}
	if issued.Credential.ID == "" {
		return fmt.Errorf("未签发凭据")
	}
	var verified struct {
		Verification struct {
			Valid  bool   `json:"valid"`
			Reason string `json:"reason"`
		} `json:"verification"`
	}
	if err := c.request(ctx, http.MethodGet, "/api/v1/public/credentials/credential-selfcheck/verify", nil, http.StatusOK, &verified); err != nil {
		return err
	}
	if !verified.Verification.Valid {
		return fmt.Errorf("凭据核验失败: %s", verified.Verification.Reason)
	}
	for _, path := range []string{
		"/api/v1/campaigns?facilityName=自检洁净厂房&pageSize=1",
		"/api/v1/campaigns/statistics?facilityName=自检洁净厂房",
		"/api/v1/campaigns/selfcheck-campaign/sampling/progress",
		"/api/v1/campaigns/selfcheck-campaign/investigations/inv-sample-alert/preflight",
		"/api/v1/campaigns/selfcheck-campaign/corrective-actions?status=completed",
		"/api/v1/campaigns/selfcheck-campaign/verifications/preflight",
		"/api/v1/campaigns/selfcheck-campaign/verifications/comparison?fromRound=1&toRound=2",
	} {
		var result map[string]any
		if err := c.request(ctx, http.MethodGet, path, nil, http.StatusOK, &result); err != nil {
			return err
		}
	}
	return nil
}

func (c *selfcheckClient) waitReady(ctx context.Context) error {
	ticker := time.NewTicker(20 * time.Millisecond)
	defer ticker.Stop()
	for {
		var body map[string]any
		if err := c.request(ctx, http.MethodGet, "/healthz", nil, http.StatusOK, &body); err == nil {
			return nil
		}
		select {
		case <-ctx.Done():
			return fmt.Errorf("等待服务就绪: %w", ctx.Err())
		case <-ticker.C:
		}
	}
}
func (c *selfcheckClient) addMeta(body map[string]any) {
	body["expectedVersion"] = c.version
	body["idempotencyKey"] = fmt.Sprintf("selfcheck-%02d", c.key)
	body["actor"] = "selfcheck"
}
func (c *selfcheckClient) writeCampaign(ctx context.Context, method, path string, body map[string]any, want string) error {
	c.addMeta(body)
	var response campaignEnvelope
	status := http.StatusOK
	if path == "/api/v1/campaigns" {
		status = http.StatusCreated
	}
	if err := c.request(ctx, method, path, body, status, &response); err != nil {
		return err
	}
	if response.Campaign.Status != want {
		return fmt.Errorf("状态应为 %s，实际为 %s", want, response.Campaign.Status)
	}
	c.version = response.Version
	return nil
}

func (c *selfcheckClient) request(ctx context.Context, method, path string, body any, want int, dst any) error {
	var reader io.Reader
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reader = bytes.NewReader(encoded)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.base+path, reader)
	if err != nil {
		return err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	payload, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if err != nil {
		return err
	}
	if resp.StatusCode != want {
		return fmt.Errorf("%s %s 返回 %d，响应 %s", method, path, resp.StatusCode, string(payload))
	}
	if dst != nil && len(payload) > 0 {
		if err := json.Unmarshal(payload, dst); err != nil {
			return fmt.Errorf("解析响应: %w", err)
		}
	}
	return nil
}

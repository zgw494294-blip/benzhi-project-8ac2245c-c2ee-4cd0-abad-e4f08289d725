package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"time"

	"subsurface-survey-gate/internal/application"
	"subsurface-survey-gate/internal/domain"
	"subsurface-survey-gate/internal/eventstore"
	"subsurface-survey-gate/internal/httpapi"
	"subsurface-survey-gate/internal/quality"
)

type selfcheckClient struct {
	base   string
	client *http.Client
}

func runSelfcheck(ctx context.Context, cfg config, logger *slog.Logger) error {
	tempDir, err := os.MkdirTemp("", "surveygate-selfcheck-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tempDir)
	store, err := eventstore.Open(tempDir)
	if err != nil {
		return err
	}
	service := application.NewService(store, quality.NewScanner())
	listener, err := net.Listen("tcp", cfg.addr)
	if err != nil {
		return err
	}
	server := newHTTPServer(listener.Addr().String(), httpapi.New(service, logger))
	done := make(chan error, 1)
	go func() { done <- server.Serve(listener) }()
	client := selfcheckClient{base: "http://" + listener.Addr().String(), client: &http.Client{Timeout: 3 * time.Second}}
	flowErr := client.fullFlow(ctx)
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	shutdownErr := server.Shutdown(shutdownCtx)
	select {
	case <-done:
	case <-shutdownCtx.Done():
		if shutdownErr == nil {
			shutdownErr = shutdownCtx.Err()
		}
	}
	if flowErr != nil {
		return flowErr
	}
	return shutdownErr
}

func (c selfcheckClient) fullFlow(ctx context.Context) error {
	now := time.Now().UTC().Add(-time.Minute)
	var campaign domain.SurveyCampaign
	if err := c.post(ctx, "/api/v1/campaigns", map[string]any{"expectedVersion": 0, "idempotencyKey": "self-create", "name": "滨江路管线探测", "surveyArea": "滨江路 K0-K1", "coordinateReference": "CGCS2000", "specificationRevision": "CJJ61-2017-r1", "actor": "selfcheck"}, http.StatusCreated, &campaign); err != nil {
		return err
	}
	var first domain.SurveyCampaign
	if err := c.post(ctx, fmt.Sprintf("/api/v1/campaigns/%s/controls", campaign.ID), map[string]any{"expectedVersion": campaign.Version, "idempotencyKey": "self-control-1", "code": "CP-001", "easting": 500000.0, "northing": 3400000.0, "elevation": 12.5, "source": "GNSS四等控制", "verifiedBy": "复核员甲", "verifiedAt": now, "actor": "selfcheck"}, http.StatusCreated, &first); err != nil {
		return err
	}
	firstID := first.Controls[0].ID
	var second domain.SurveyCampaign
	if err := c.post(ctx, fmt.Sprintf("/api/v1/campaigns/%s/controls", campaign.ID), map[string]any{"expectedVersion": first.Version, "idempotencyKey": "self-control-2", "code": "CP-002", "easting": 500100.0, "northing": 3400100.0, "elevation": 12.7, "source": "GNSS四等控制", "verifiedBy": "复核员甲", "verifiedAt": now, "actor": "selfcheck"}, http.StatusCreated, &second); err != nil {
		return err
	}
	secondID := second.Controls[1].ID
	var locked domain.SurveyCampaign
	if err := c.post(ctx, fmt.Sprintf("/api/v1/campaigns/%s/baseline/lock", campaign.ID), map[string]any{"expectedVersion": second.Version, "idempotencyKey": "self-lock", "actor": "selfcheck"}, http.StatusOK, &locked); err != nil {
		return err
	}
	var observed domain.SurveyCampaign
	if err := c.post(ctx, fmt.Sprintf("/api/v1/campaigns/%s/observations", campaign.ID), map[string]any{"expectedVersion": locked.Version, "idempotencyKey": "self-observation", "segmentCode": "SEG-001", "utilityType": "给水", "startPointId": firstID, "endPointId": secondID, "burialDepthMm": 1800, "diameterMm": 300, "material": "球墨铸铁", "detectionMethod": "电磁感应+开井验证", "observedAt": now, "actor": "selfcheck"}, http.StatusCreated, &observed); err != nil {
		return err
	}
	var scanned domain.SurveyCampaign
	if err := c.post(ctx, fmt.Sprintf("/api/v1/campaigns/%s/scans", campaign.ID), map[string]any{"expectedVersion": observed.Version, "idempotencyKey": "self-scan", "actor": "selfcheck"}, http.StatusOK, &scanned); err != nil {
		return err
	}
	if scanned.State != domain.StateReadyForReview {
		return fmt.Errorf("扫描后状态异常: %s", scanned.State)
	}
	var reviewing domain.SurveyCampaign
	if err := c.post(ctx, fmt.Sprintf("/api/v1/campaigns/%s/review/submit", campaign.ID), map[string]any{"expectedVersion": scanned.Version, "idempotencyKey": "self-submit", "submitter": "作业负责人"}, http.StatusOK, &reviewing); err != nil {
		return err
	}
	var approved domain.SurveyCampaign
	if err := c.post(ctx, fmt.Sprintf("/api/v1/campaigns/%s/review/decision", campaign.ID), map[string]any{"expectedVersion": reviewing.Version, "idempotencyKey": "self-approve", "reviewer": "质量复核员", "decision": "approve", "reason": "成果满足规范"}, http.StatusOK, &approved); err != nil {
		return err
	}
	var frozen domain.SurveyCampaign
	if err := c.post(ctx, fmt.Sprintf("/api/v1/campaigns/%s/freeze", campaign.ID), map[string]any{"expectedVersion": approved.Version, "idempotencyKey": "self-freeze", "actor": "质量复核员"}, http.StatusOK, &frozen); err != nil {
		return err
	}
	var credential domain.ReleaseCredential
	if err := c.post(ctx, fmt.Sprintf("/api/v1/campaigns/%s/credential", campaign.ID), map[string]any{"expectedVersion": frozen.Version, "idempotencyKey": "self-issue", "issuedBy": "市政测绘质量中心"}, http.StatusCreated, &credential); err != nil {
		return err
	}
	var verification domain.CredentialVerification
	if err := c.post(ctx, "/api/v1/credentials/verify", map[string]any{"credential": credential}, http.StatusOK, &verification); err != nil {
		return err
	}
	if !verification.Valid || verification.FrozenVersion != frozen.Version {
		return fmt.Errorf("凭据核验未通过: %s", verification.Reason)
	}
	return nil
}

func (c selfcheckClient) post(ctx context.Context, path string, payload any, expected int, dst any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.base+path, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if err != nil {
		return err
	}
	if resp.StatusCode != expected {
		return fmt.Errorf("%s 返回 %d，期望 %d: %s", path, resp.StatusCode, expected, string(data))
	}
	if err := json.Unmarshal(data, dst); err != nil {
		return fmt.Errorf("解析 %s 响应: %w", path, err)
	}
	return nil
}

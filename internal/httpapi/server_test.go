package httpapi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"cleanroom-release-go/internal/ledger"
	"cleanroom-release-go/internal/workflow"
)

func testServer(t *testing.T) *httptest.Server {
	store, err := ledger.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	return httptest.NewServer(New(workflow.New(store, "secret"), nil).Handler())
}

func TestHealthAndValidationErrorShape(t *testing.T) {
	server := testServer(t)
	defer server.Close()
	resp, err := http.Get(server.URL + "/healthz")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatal(resp.Status)
	}
	body := []byte(`{"id":"c1","facilityName":"厂房","sites":[]}`)
	resp, err = http.Post(server.URL+"/api/v1/campaigns", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("状态=%d", resp.StatusCode)
	}
	var result struct {
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatal(err)
	}
	if result.Error.Code == "" || result.Error.Message == "" {
		t.Fatalf("错误响应不可机器读取: %+v", result)
	}
}

func TestUnknownJSONFieldRejected(t *testing.T) {
	server := testServer(t)
	defer server.Close()
	body := []byte(`{"id":"c1","facilityName":"厂房","unexpected":true,"sites":[]}`)
	resp, err := http.Post(server.URL+"/api/v1/campaigns", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("未知字段应返回 400，实际 %d", resp.StatusCode)
	}
}

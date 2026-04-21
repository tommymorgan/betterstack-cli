package client

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestClient_SendsCorrectAuthHeaders(t *testing.T) {
	var gotAuth, gotAccept, gotUA string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotAccept = r.Header.Get("Accept")
		gotUA = r.Header.Get("User-Agent")
		fmt.Fprint(w, `{"data":[]}`)
	}))
	defer srv.Close()

	c := New("test-token", "1.0.0").withBaseURL(srv.URL)
	_, err := c.ListIncidents(context.Background(), IncidentListParams{}, 0)
	if err != nil {
		t.Fatal(err)
	}

	if gotAuth != "Bearer test-token" {
		t.Errorf("Authorization = %q, want %q", gotAuth, "Bearer test-token")
	}
	if gotAccept != "application/json" {
		t.Errorf("Accept = %q, want %q", gotAccept, "application/json")
	}
	if gotUA != "betterstack-cli/1.0.0" {
		t.Errorf("User-Agent = %q, want %q", gotUA, "betterstack-cli/1.0.0")
	}
}

func TestClient_AutoPaginates(t *testing.T) {
	callCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		if callCount == 1 {
			nextURL := fmt.Sprintf("http://%s/page2", r.Host)
			fmt.Fprintf(w, `{"data":[{"id":"1"}],"pagination":{"next":"%s"}}`, nextURL)
		} else {
			fmt.Fprint(w, `{"data":[{"id":"2"}],"pagination":{"next":null}}`)
		}
	}))
	defer srv.Close()

	c := New("token", "dev").withBaseURL(srv.URL)
	results, err := c.ListIncidents(context.Background(), IncidentListParams{}, 0)
	if err != nil {
		t.Fatal(err)
	}

	if len(results) != 2 {
		t.Errorf("got %d results, want 2", len(results))
	}
	if callCount != 2 {
		t.Errorf("expected 2 requests, got %d", callCount)
	}
}

func TestClient_RespectsLimit(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextURL := fmt.Sprintf("http://%s/page2", r.Host)
		fmt.Fprintf(w, `{"data":[{"id":"1"},{"id":"2"},{"id":"3"}],"pagination":{"next":"%s"}}`, nextURL)
	}))
	defer srv.Close()

	c := New("token", "dev").withBaseURL(srv.URL)
	results, err := c.ListIncidents(context.Background(), IncidentListParams{}, 2)
	if err != nil {
		t.Fatal(err)
	}

	if len(results) != 2 {
		t.Errorf("got %d results, want 2", len(results))
	}
}

func TestClient_RetriesOn429(t *testing.T) {
	attempts := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts < 3 {
			w.Header().Set("Retry-After", "0")
			w.WriteHeader(http.StatusTooManyRequests)
			fmt.Fprint(w, `{"error":"rate limited"}`)
			return
		}
		fmt.Fprint(w, `{"data":[{"id":"1"}]}`)
	}))
	defer srv.Close()

	c := New("token", "dev").withBaseURL(srv.URL)
	results, err := c.ListIncidents(context.Background(), IncidentListParams{}, 0)
	if err != nil {
		t.Fatal(err)
	}

	if len(results) != 1 {
		t.Errorf("got %d results, want 1", len(results))
	}
	if attempts != 3 {
		t.Errorf("expected 3 attempts, got %d", attempts)
	}
}

func TestClient_ReturnsErrorAfterMaxRetries(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Retry-After", "0")
		w.WriteHeader(http.StatusTooManyRequests)
		fmt.Fprint(w, `{"error":"rate limited"}`)
	}))
	defer srv.Close()

	c := New("token", "dev").withBaseURL(srv.URL)
	_, err := c.ListIncidents(context.Background(), IncidentListParams{}, 0)
	if err == nil {
		t.Fatal("expected error after max retries")
	}
}

func TestClient_ReturnsAPIErrorForNon2xx(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprint(w, `{"errors":["unauthorized"]}`)
	}))
	defer srv.Close()

	c := New("token", "dev").withBaseURL(srv.URL)
	_, err := c.ListIncidents(context.Background(), IncidentListParams{}, 0)
	if err == nil {
		t.Fatal("expected error for 401")
	}

	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("expected *APIError, got %T", err)
	}
	if apiErr.StatusCode != 401 {
		t.Errorf("got status %d, want 401", apiErr.StatusCode)
	}
}

func TestClient_EnforcesRequestTimeout(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		fmt.Fprint(w, `{"data":[]}`)
	}))
	defer srv.Close()

	c := New("token", "dev").withBaseURL(srv.URL)
	c.httpClientField().Timeout = 100 * time.Millisecond

	_, err := c.ListIncidents(context.Background(), IncidentListParams{}, 0)
	if err == nil {
		t.Fatal("expected timeout error")
	}
	if !strings.Contains(err.Error(), "deadline exceeded") && !strings.Contains(err.Error(), "Timeout") && !strings.Contains(err.Error(), "timeout") {
		t.Errorf("expected timeout error, got: %v", err)
	}
}

func TestClient_IgnoresUnknownJSONFields(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"data":[{"id":"1","unknown_field":"value","nested":{"deep":"data"}}]}`)
	}))
	defer srv.Close()

	c := New("token", "dev").withBaseURL(srv.URL)
	results, err := c.ListIncidents(context.Background(), IncidentListParams{}, 0)
	if err != nil {
		t.Fatal(err)
	}

	if len(results) != 1 {
		t.Fatalf("got %d results, want 1", len(results))
	}

	// Verify the raw JSON is preserved including unknown fields
	var m map[string]interface{}
	if err := json.Unmarshal(results[0], &m); err != nil {
		t.Fatal(err)
	}
	if m["id"] != "1" {
		t.Errorf("known field 'id' = %v, want '1'", m["id"])
	}
}

func TestClient_ListIncidentsSendsCorrectQueryParams(t *testing.T) {
	var gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		fmt.Fprint(w, `{"data":[]}`)
	}))
	defer srv.Close()

	resolved := true
	c := New("token", "dev").withBaseURL(srv.URL)
	_, err := c.ListIncidents(context.Background(), IncidentListParams{
		From:      "2026-03-13",
		To:        "2026-03-20",
		MonitorID: "123",
		Resolved:  &resolved,
	}, 0)
	if err != nil {
		t.Fatal(err)
	}

	for _, expected := range []string{"from=2026-03-13", "to=2026-03-20", "monitor_id=123", "resolved=true"} {
		if !strings.Contains(gotQuery, expected) {
			t.Errorf("query %q missing %q", gotQuery, expected)
		}
	}
}

func TestClient_ListIncidentsSendsToCorrectEndpoint(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		fmt.Fprint(w, `{"data":[]}`)
	}))
	defer srv.Close()

	c := New("token", "dev").withBaseURL(srv.URL)
	_, _ = c.ListIncidents(context.Background(), IncidentListParams{}, 0)

	if gotPath != incidentsPath {
		t.Errorf("path = %q, want %q", gotPath, incidentsPath)
	}
}

func TestClient_GetIncidentSendsToCorrectEndpoint(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		fmt.Fprint(w, `{"data":{"id":"42"}}`)
	}))
	defer srv.Close()

	c := New("token", "dev").withBaseURL(srv.URL)
	_, _ = c.GetIncident(context.Background(), "42")

	if gotPath != incidentsPath+"/42" {
		t.Errorf("path = %q, want %q", gotPath, incidentsPath+"/42")
	}
}

func TestClient_ListMonitorsSendsToCorrectEndpoint(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		fmt.Fprint(w, `{"data":[]}`)
	}))
	defer srv.Close()

	c := New("token", "dev").withBaseURL(srv.URL)
	_, _ = c.ListMonitors(context.Background(), MonitorListParams{}, 0)

	if gotPath != monitorsPath {
		t.Errorf("path = %q, want %q", gotPath, monitorsPath)
	}
}

func TestClient_GetMonitorSendsToCorrectEndpoint(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		fmt.Fprint(w, `{"data":{"id":"99"}}`)
	}))
	defer srv.Close()

	c := New("token", "dev").withBaseURL(srv.URL)
	_, _ = c.GetMonitor(context.Background(), "99")

	if gotPath != monitorsPath+"/99" {
		t.Errorf("path = %q, want %q", gotPath, monitorsPath+"/99")
	}
}

func TestClient_PostRequestSendsContentTypeJSON(t *testing.T) {
	var gotMethod, gotContentType string
	var gotBody []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotContentType = r.Header.Get("Content-Type")
		gotBody, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusCreated)
		fmt.Fprint(w, `{"data":{"id":"1","attributes":{"name":"test"}}}`)
	}))
	defer srv.Close()

	c := New("token", "dev").withBaseURL(srv.URL)
	body := strings.NewReader(`{"name":"test","type":"amazon_cloudwatch"}`)
	result, err := c.t.postOne(context.Background(), srv.URL+"/api/v2/sources", body)
	if err != nil {
		t.Fatal(err)
	}

	if gotMethod != "POST" {
		t.Errorf("method = %q, want POST", gotMethod)
	}
	if gotContentType != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", gotContentType)
	}
	if !strings.Contains(string(gotBody), `"name"`) {
		t.Errorf("body missing name field: %s", gotBody)
	}
	if result == nil {
		t.Error("expected non-nil result")
	}
}

func TestClient_DeleteRequestSendsNoBody(t *testing.T) {
	var gotMethod string
	var gotBodyLen int64
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotBodyLen = r.ContentLength
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	c := New("token", "dev").withBaseURL(srv.URL)
	err := c.t.deleteOne(context.Background(), srv.URL+"/api/v2/sources/1")
	if err != nil {
		t.Fatal(err)
	}

	if gotMethod != "DELETE" {
		t.Errorf("method = %q, want DELETE", gotMethod)
	}
	if gotBodyLen > 0 {
		t.Errorf("DELETE should have no body, got Content-Length %d", gotBodyLen)
	}
}

func TestClient_ListIntegrationsSendsToCorrectEndpoint(t *testing.T) {
	var gotPath, gotMethod string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotMethod = r.Method
		fmt.Fprint(w, `{"data":[{"id":"1","attributes":{"name":"CloudWatch","webhook_url":"https://example.com/hook","paused":false}}]}`)
	}))
	defer srv.Close()

	c := New("token", "dev").withBaseURL(srv.URL)
	results, err := c.ListIntegrations(context.Background(), "aws-cloudwatch", 0)
	if err != nil {
		t.Fatal(err)
	}

	if gotMethod != "GET" {
		t.Errorf("method = %q, want GET", gotMethod)
	}
	if gotPath != "/api/v2/aws-cloudwatch-integrations" {
		t.Errorf("path = %q, want /api/v2/aws-cloudwatch-integrations", gotPath)
	}
	if len(results) != 1 {
		t.Errorf("got %d results, want 1", len(results))
	}
}

func TestClient_GetIntegrationSendsToCorrectEndpoint(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		fmt.Fprint(w, `{"data":{"id":"42","attributes":{"name":"CloudWatch","webhook_url":"https://example.com/hook","paused":false}}}`)
	}))
	defer srv.Close()

	c := New("token", "dev").withBaseURL(srv.URL)
	result, err := c.GetIntegration(context.Background(), "aws-cloudwatch", "42")
	if err != nil {
		t.Fatal(err)
	}

	if gotPath != "/api/v2/aws-cloudwatch-integrations/42" {
		t.Errorf("path = %q, want /api/v2/aws-cloudwatch-integrations/42", gotPath)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal(result, &parsed); err != nil {
		t.Fatal(err)
	}
	if parsed["id"] != "42" {
		t.Errorf("id = %v, want 42", parsed["id"])
	}
}

func TestClient_CreateIntegrationSendsPostWithJSONBody(t *testing.T) {
	var gotPath, gotMethod, gotContentType string
	var gotBody map[string]string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotMethod = r.Method
		gotContentType = r.Header.Get("Content-Type")
		bodyBytes, _ := io.ReadAll(r.Body)
		if err := json.Unmarshal(bodyBytes, &gotBody); err != nil {
			t.Errorf("failed to parse request body: %v", err)
		}
		w.WriteHeader(http.StatusCreated)
		fmt.Fprint(w, `{"data":{"id":"1","attributes":{"name":"dev-cw","webhook_url":"https://uptime.betterstack.com/hook/abc123","paused":false}}}`)
	}))
	defer srv.Close()

	c := New("token", "dev").withBaseURL(srv.URL)
	result, err := c.CreateIntegration(context.Background(), "aws-cloudwatch", "dev-cw")
	if err != nil {
		t.Fatal(err)
	}

	if gotMethod != "POST" {
		t.Errorf("method = %q, want POST", gotMethod)
	}
	if gotPath != "/api/v2/aws-cloudwatch-integrations" {
		t.Errorf("path = %q, want /api/v2/aws-cloudwatch-integrations", gotPath)
	}
	if gotContentType != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", gotContentType)
	}
	if gotBody["name"] != "dev-cw" {
		t.Errorf("body name = %q, want dev-cw", gotBody["name"])
	}
	if result == nil {
		t.Error("expected non-nil result")
	}
}

func TestClient_DeleteIntegrationSendsDeleteAndSucceedsOn204(t *testing.T) {
	var gotPath, gotMethod string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotMethod = r.Method
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	c := New("token", "dev").withBaseURL(srv.URL)
	err := c.DeleteIntegration(context.Background(), "aws-cloudwatch", "42")
	if err != nil {
		t.Fatal(err)
	}

	if gotMethod != "DELETE" {
		t.Errorf("method = %q, want DELETE", gotMethod)
	}
	if gotPath != "/api/v2/aws-cloudwatch-integrations/42" {
		t.Errorf("path = %q, want /api/v2/aws-cloudwatch-integrations/42", gotPath)
	}
}

func TestClient_UnknownIntegrationTypeReturnsError(t *testing.T) {
	c := New("token", "dev")

	_, err := c.ListIntegrations(context.Background(), "nonexistent", 0)
	if err == nil {
		t.Fatal("expected error for unknown integration type")
	}
	if !strings.Contains(err.Error(), "nonexistent") {
		t.Errorf("error should name the invalid type, got: %s", err.Error())
	}
	if !strings.Contains(err.Error(), "aws-cloudwatch") {
		t.Errorf("error should list supported types, got: %s", err.Error())
	}
}

func TestClient_DeleteIntegrationReturnsErrorOn404(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprint(w, `{"errors":["not found"]}`)
	}))
	defer srv.Close()

	c := New("token", "dev").withBaseURL(srv.URL)
	err := c.DeleteIntegration(context.Background(), "aws-cloudwatch", "999")
	if err == nil {
		t.Fatal("expected error for 404")
	}

	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("expected *APIError, got %T", err)
	}
	if apiErr.StatusCode != 404 {
		t.Errorf("got status %d, want 404", apiErr.StatusCode)
	}
}

func TestClient_ExistingGetCallersUnchanged(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("expected GET, got %s", r.Method)
		}
		fmt.Fprint(w, `{"data":[{"id":"1"}]}`)
	}))
	defer srv.Close()

	c := New("token", "dev").withBaseURL(srv.URL)
	results, err := c.ListIncidents(context.Background(), IncidentListParams{}, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 {
		t.Errorf("got %d results, want 1", len(results))
	}
}

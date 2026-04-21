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

func TestTelemetry_UsesTelemetryBaseURL(t *testing.T) {
	t.Setenv("BETTERSTACK_TELEMETRY_BASE_URL", "")
	c := NewTelemetry("tok", "dev")
	if c.t.baseURL != telemetryBaseURL {
		t.Errorf("baseURL = %q, want %q", c.t.baseURL, telemetryBaseURL)
	}
}

func TestTelemetry_HonorsBaseURLEnvOverride(t *testing.T) {
	t.Setenv("BETTERSTACK_TELEMETRY_BASE_URL", "http://localhost:9999")
	c := NewTelemetry("tok", "dev")
	if c.t.baseURL != "http://localhost:9999" {
		t.Errorf("baseURL = %q, want override", c.t.baseURL)
	}
}

func TestTelemetry_SetsBearerAuthHeader(t *testing.T) {
	var gotAuth, gotAccept, gotUA string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotAccept = r.Header.Get("Accept")
		gotUA = r.Header.Get("User-Agent")
		fmt.Fprint(w, `{"data":[]}`)
	}))
	defer srv.Close()

	c := NewTelemetry("telemetry-token", "2.0.0").withBaseURL(srv.URL)
	_, err := c.ListExplorations(context.Background(), 0)
	if err != nil {
		t.Fatal(err)
	}
	if gotAuth != "Bearer telemetry-token" {
		t.Errorf("Authorization = %q, want Bearer telemetry-token", gotAuth)
	}
	if gotAccept != "application/json" {
		t.Errorf("Accept = %q, want application/json", gotAccept)
	}
	if !strings.Contains(gotUA, "betterstack-cli/") {
		t.Errorf("User-Agent = %q, want betterstack-cli/*", gotUA)
	}
}

func TestTelemetry_EmitsSameAPIErrorTypeAsUptime(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprint(w, `{"errors":["bad token"]}`)
	}))
	defer srv.Close()

	c := NewTelemetry("tok", "dev").withBaseURL(srv.URL)
	_, err := c.ListExplorations(context.Background(), 0)
	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("expected *APIError, got %T", err)
	}
	if apiErr.StatusCode != 401 {
		t.Errorf("got %d, want 401", apiErr.StatusCode)
	}
}

func TestTelemetry_RetriesOn429WithRetryAfter(t *testing.T) {
	attempts := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts < 3 {
			w.Header().Set("Retry-After", "0")
			w.WriteHeader(http.StatusTooManyRequests)
			fmt.Fprint(w, `{"error":"rate limit"}`)
			return
		}
		fmt.Fprint(w, `{"data":[{"id":"1"}]}`)
	}))
	defer srv.Close()

	c := NewTelemetry("tok", "dev").withBaseURL(srv.URL)
	results, err := c.ListExplorations(context.Background(), 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 {
		t.Errorf("got %d results, want 1", len(results))
	}
	if attempts != 3 {
		t.Errorf("attempts = %d, want 3", attempts)
	}
}

func TestTelemetry_EnforcesRequestTimeout(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		fmt.Fprint(w, `{"data":[]}`)
	}))
	defer srv.Close()

	c := NewTelemetry("tok", "dev").withBaseURL(srv.URL)
	c.httpClientField().Timeout = 100 * time.Millisecond

	_, err := c.ListExplorations(context.Background(), 0)
	if err == nil {
		t.Fatal("expected timeout error")
	}
}

func TestTelemetry_ExplorationEndpoints(t *testing.T) {
	type req struct {
		method, path string
		body         []byte
	}
	var gotReqs []req
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		gotReqs = append(gotReqs, req{r.Method, r.URL.Path, body})
		switch r.Method {
		case "GET":
			if strings.HasSuffix(r.URL.Path, "/123") {
				fmt.Fprint(w, `{"data":{"id":"123"}}`)
			} else {
				fmt.Fprint(w, `{"data":[{"id":"1"}]}`)
			}
		case "POST":
			w.WriteHeader(http.StatusCreated)
			fmt.Fprint(w, `{"data":{"id":"new"}}`)
		case "PATCH":
			fmt.Fprint(w, `{"data":{"id":"123"}}`)
		case "DELETE":
			w.WriteHeader(http.StatusNoContent)
		}
	}))
	defer srv.Close()

	c := NewTelemetry("tok", "dev").withBaseURL(srv.URL)
	ctx := context.Background()

	if _, err := c.ListExplorations(ctx, 0); err != nil {
		t.Fatal(err)
	}
	if _, err := c.GetExploration(ctx, "123"); err != nil {
		t.Fatal(err)
	}
	if _, err := c.CreateExploration(ctx, []byte(`{"name":"x"}`)); err != nil {
		t.Fatal(err)
	}
	if _, err := c.UpdateExploration(ctx, "123", []byte(`{"name":"y"}`)); err != nil {
		t.Fatal(err)
	}
	if err := c.DeleteExploration(ctx, "123"); err != nil {
		t.Fatal(err)
	}

	expected := []req{
		{"GET", "/api/v2/explorations", nil},
		{"GET", "/api/v2/explorations/123", nil},
		{"POST", "/api/v2/explorations", []byte(`{"name":"x"}`)},
		{"PATCH", "/api/v2/explorations/123", []byte(`{"name":"y"}`)},
		{"DELETE", "/api/v2/explorations/123", nil},
	}
	for i, want := range expected {
		if gotReqs[i].method != want.method || gotReqs[i].path != want.path {
			t.Errorf("request %d: got %s %s, want %s %s", i, gotReqs[i].method, gotReqs[i].path, want.method, want.path)
		}
		if want.body != nil && string(gotReqs[i].body) != string(want.body) {
			t.Errorf("request %d body: got %q, want %q", i, gotReqs[i].body, want.body)
		}
	}
}

func TestTelemetry_AlertsEndpoints(t *testing.T) {
	type req struct {
		method, path string
		body         []byte
	}
	var gotReqs []req
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		gotReqs = append(gotReqs, req{r.Method, r.URL.Path, body})
		switch r.Method {
		case "GET":
			if strings.Contains(r.URL.Path, "/alerts/") {
				fmt.Fprint(w, `{"data":{"id":"99"}}`)
			} else {
				fmt.Fprint(w, `{"data":[]}`)
			}
		case "POST":
			w.WriteHeader(http.StatusCreated)
			fmt.Fprint(w, `{"data":{"id":"new"}}`)
		case "PATCH":
			fmt.Fprint(w, `{"data":{"id":"99"}}`)
		case "DELETE":
			w.WriteHeader(http.StatusNoContent)
		}
	}))
	defer srv.Close()

	c := NewTelemetry("tok", "dev").withBaseURL(srv.URL)
	ctx := context.Background()

	_, _ = c.ListAlerts(ctx, 0)
	_, _ = c.GetAlert(ctx, "99")
	_, _ = c.CreateAlert(ctx, "5", []byte(`{"threshold":3}`))
	_, _ = c.UpdateAlert(ctx, "99", []byte(`{"name":"u"}`))
	_ = c.DeleteAlert(ctx, "99")

	expected := []req{
		{"GET", "/api/v2/alerts", nil},
		{"GET", "/api/v2/alerts/99", nil},
		{"POST", "/api/v2/explorations/5/alerts", []byte(`{"threshold":3}`)},
		{"PATCH", "/api/v2/alerts/99", []byte(`{"name":"u"}`)},
		{"DELETE", "/api/v2/alerts/99", nil},
	}
	for i, want := range expected {
		if gotReqs[i].method != want.method || gotReqs[i].path != want.path {
			t.Errorf("request %d: got %s %s, want %s %s", i, gotReqs[i].method, gotReqs[i].path, want.method, want.path)
		}
	}
}

func TestTelemetry_SourcesEndpoints(t *testing.T) {
	var paths []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path)
		if strings.HasSuffix(r.URL.Path, "/42") {
			fmt.Fprint(w, `{"data":{"id":"42"}}`)
		} else {
			fmt.Fprint(w, `{"data":[]}`)
		}
	}))
	defer srv.Close()

	c := NewTelemetry("tok", "dev").withBaseURL(srv.URL)
	ctx := context.Background()

	_, _ = c.ListSources(ctx, 0)
	_, _ = c.GetSource(ctx, "42")

	want := []string{"/api/v2/sources", "/api/v2/sources/42"}
	for i, p := range want {
		if paths[i] != p {
			t.Errorf("path %d = %q, want %q", i, paths[i], p)
		}
	}
}

func TestTelemetry_ExplorationAlertsFirstPage_ShortCircuitsPagination(t *testing.T) {
	var gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		fmt.Fprint(w, `{"data":[{"id":"9"}],"pagination":{"next":"https://next.example/never"}}`)
	}))
	defer srv.Close()

	c := NewTelemetry("tok", "dev").withBaseURL(srv.URL)
	results, err := c.ExplorationAlertsFirstPage(context.Background(), "55", 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 {
		t.Errorf("got %d results, want 1", len(results))
	}
	if !strings.Contains(gotQuery, "per_page=1") {
		t.Errorf("query missing per_page=1: %s", gotQuery)
	}
}

func TestTelemetry_IterateAlerts_ShortCircuitsOnMatch(t *testing.T) {
	var pageCount int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		pageCount++
		if pageCount == 1 {
			next := fmt.Sprintf("http://%s/page2", r.Host)
			fmt.Fprintf(w, `{"data":[{"id":"1"},{"id":"2"}],"pagination":{"next":"%s"}}`, next)
			return
		}
		t.Error("second page should not be fetched")
	}))
	defer srv.Close()

	c := NewTelemetry("tok", "dev").withBaseURL(srv.URL)
	match, err := c.IterateAlerts(context.Background(), func(item json.RawMessage) bool {
		var m map[string]string
		_ = json.Unmarshal(item, &m)
		return m["id"] == "2"
	})
	if err != nil {
		t.Fatal(err)
	}
	if match == nil {
		t.Fatal("expected match")
	}
	if pageCount != 1 {
		t.Errorf("pageCount = %d, want 1", pageCount)
	}
}

func TestUptime_PoliciesEndpoints(t *testing.T) {
	var paths []string
	var methods []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path)
		methods = append(methods, r.Method)
		switch r.Method {
		case "GET":
			if strings.HasSuffix(r.URL.Path, "/7") {
				fmt.Fprint(w, `{"data":{"id":"7"}}`)
			} else {
				fmt.Fprint(w, `{"data":[]}`)
			}
		case "POST":
			w.WriteHeader(http.StatusCreated)
			fmt.Fprint(w, `{"data":{"id":"new"}}`)
		case "PATCH":
			fmt.Fprint(w, `{"data":{"id":"7"}}`)
		case "DELETE":
			w.WriteHeader(http.StatusNoContent)
		}
	}))
	defer srv.Close()

	c := New("tok", "dev").withBaseURL(srv.URL)
	ctx := context.Background()
	_, _ = c.ListPolicies(ctx, 0)
	_, _ = c.GetPolicy(ctx, "7")
	_, _ = c.CreatePolicy(ctx, []byte(`{"name":"p"}`))
	_, _ = c.UpdatePolicy(ctx, "7", []byte(`{"name":"q"}`))
	_ = c.DeletePolicy(ctx, "7")

	wantPaths := []string{
		"/api/v3/policies",
		"/api/v3/policies/7",
		"/api/v3/policies",
		"/api/v3/policies/7",
		"/api/v3/policies/7",
	}
	wantMethods := []string{"GET", "GET", "POST", "PATCH", "DELETE"}
	for i := range wantPaths {
		if paths[i] != wantPaths[i] {
			t.Errorf("path %d = %q, want %q", i, paths[i], wantPaths[i])
		}
		if methods[i] != wantMethods[i] {
			t.Errorf("method %d = %q, want %q", i, methods[i], wantMethods[i])
		}
	}
}

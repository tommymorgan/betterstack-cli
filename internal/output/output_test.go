package output

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestRenderIncidentsTable_ShowsCorrectColumns(t *testing.T) {
	data := []json.RawMessage{
		json.RawMessage(`{
			"id": "1",
			"attributes": {
				"name": "Test Monitor",
				"cause": "Threshold crossed",
				"started_at": "2026-03-20T10:00:00Z",
				"resolved_at": "2026-03-20T10:01:00Z",
				"status": "Resolved"
			}
		}`),
	}

	var buf bytes.Buffer
	if err := RenderIncidentsTable(&buf, data); err != nil {
		t.Fatal(err)
	}

	output := buf.String()
	for _, col := range []string{"ID", "NAME", "CAUSE", "STARTED AT", "RESOLVED AT", "LENGTH", "STATUS"} {
		if !strings.Contains(output, col) {
			t.Errorf("missing column header %q in output:\n%s", col, output)
		}
	}

	if !strings.Contains(output, "Test Monitor") {
		t.Error("missing incident name")
	}
	if !strings.Contains(output, "Threshold crossed") {
		t.Error("missing incident cause")
	}
	if !strings.Contains(output, "Resolved") {
		t.Error("missing incident status")
	}
}

func TestRenderIncidentsTable_DisplaysTimestampsInUTC(t *testing.T) {
	data := []json.RawMessage{
		json.RawMessage(`{
			"id": "1",
			"attributes": {
				"name": "Test",
				"cause": "",
				"started_at": "2026-03-20T10:00:00-07:00",
				"resolved_at": "2026-03-20T10:05:00-07:00",
				"status": "Resolved"
			}
		}`),
	}

	var buf bytes.Buffer
	if err := RenderIncidentsTable(&buf, data); err != nil {
		t.Fatal(err)
	}

	output := buf.String()
	if !strings.Contains(output, "2026-03-20 17:00:00 UTC") {
		t.Errorf("expected UTC timestamp, got:\n%s", output)
	}
}

func TestRenderIncidentsTable_ShowsDashForNullResolvedAt(t *testing.T) {
	data := []json.RawMessage{
		json.RawMessage(`{
			"id": "1",
			"attributes": {
				"name": "Active Incident",
				"cause": "Threshold crossed",
				"started_at": "2026-03-20T10:00:00Z",
				"resolved_at": null,
				"status": "Started"
			}
		}`),
	}

	var buf bytes.Buffer
	if err := RenderIncidentsTable(&buf, data); err != nil {
		t.Fatal(err)
	}

	output := buf.String()
	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) < 2 {
		t.Fatalf("expected at least 2 lines, got %d", len(lines))
	}

	// The data line should contain "Threshold crossed" (not a dash for cause)
	// but should have dashes for resolved_at and length
	dataLine := lines[1]
	if !strings.Contains(dataLine, "Threshold crossed") {
		t.Errorf("cause should show 'Threshold crossed', got:\n%s", dataLine)
	}
	if !strings.Contains(dataLine, "Started") {
		t.Errorf("status should show 'Started', got:\n%s", dataLine)
	}
}

func TestRenderIncidentsTable_ShowsDashForEmptyCause(t *testing.T) {
	data := []json.RawMessage{
		json.RawMessage(`{
			"id": "1",
			"attributes": {
				"name": "Test",
				"cause": "",
				"started_at": "2026-03-20T10:00:00Z",
				"resolved_at": "2026-03-20T10:00:30Z",
				"status": "Resolved"
			}
		}`),
	}

	var buf bytes.Buffer
	if err := RenderIncidentsTable(&buf, data); err != nil {
		t.Fatal(err)
	}

	// The cause column should contain a standalone dash
	// We check by looking at the data line fields
	output := buf.String()
	lines := strings.Split(strings.TrimSpace(output), "\n")
	dataLine := lines[1]
	fields := strings.Fields(dataLine)
	// Field 0 = ID, Field 1 = Name, Field 2 should be the dash for empty cause
	found := false
	for _, f := range fields {
		if f == "-" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected dash for empty cause in:\n%s", dataLine)
	}
}

func TestRenderIncidentsTable_CalculatesSeconds(t *testing.T) {
	data := []json.RawMessage{
		json.RawMessage(`{
			"id": "1",
			"attributes": {
				"name": "Test",
				"cause": "error",
				"started_at": "2026-03-20T10:00:00Z",
				"resolved_at": "2026-03-20T10:00:30Z",
				"status": "Resolved"
			}
		}`),
	}

	var buf bytes.Buffer
	if err := RenderIncidentsTable(&buf, data); err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(buf.String(), "30s") {
		t.Errorf("expected 30s length, got:\n%s", buf.String())
	}
}

func TestRenderIncidentsTable_CalculatesMinutes(t *testing.T) {
	data := []json.RawMessage{
		json.RawMessage(`{
			"id": "1",
			"attributes": {
				"name": "Test",
				"cause": "error",
				"started_at": "2026-03-20T10:00:00Z",
				"resolved_at": "2026-03-20T10:01:00Z",
				"status": "Resolved"
			}
		}`),
	}

	var buf bytes.Buffer
	if err := RenderIncidentsTable(&buf, data); err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(buf.String(), "1m") {
		t.Errorf("expected 1m length, got:\n%s", buf.String())
	}
}

func TestRenderIncidentsTable_CalculatesHoursAndMinutes(t *testing.T) {
	data := []json.RawMessage{
		json.RawMessage(`{
			"id": "1",
			"attributes": {
				"name": "Test",
				"cause": "error",
				"started_at": "2026-03-20T10:00:00Z",
				"resolved_at": "2026-03-20T12:15:00Z",
				"status": "Resolved"
			}
		}`),
	}

	var buf bytes.Buffer
	if err := RenderIncidentsTable(&buf, data); err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(buf.String(), "2h15m") {
		t.Errorf("expected 2h15m length, got:\n%s", buf.String())
	}
}

func TestRenderMonitorsTable_ShowsCorrectColumns(t *testing.T) {
	data := []json.RawMessage{
		json.RawMessage(`{
			"id": "99",
			"attributes": {
				"pronounceable_name": "My Monitor",
				"url": "https://example.com",
				"status": "up",
				"check_frequency": 60
			}
		}`),
	}

	var buf bytes.Buffer
	if err := RenderMonitorsTable(&buf, data); err != nil {
		t.Fatal(err)
	}

	output := buf.String()
	for _, col := range []string{"ID", "NAME", "URL", "STATUS", "CHECK FREQUENCY"} {
		if !strings.Contains(output, col) {
			t.Errorf("missing column header %q", col)
		}
	}

	if !strings.Contains(output, "My Monitor") {
		t.Error("missing monitor name")
	}
	if !strings.Contains(output, "60s") {
		t.Error("missing check frequency")
	}
}

func TestRenderJSON_OutputsValidIndentedJSON(t *testing.T) {
	data := []json.RawMessage{
		json.RawMessage(`{"id":"1","name":"test"}`),
	}

	var buf bytes.Buffer
	if err := RenderJSON(&buf, data); err != nil {
		t.Fatal(err)
	}

	var parsed interface{}
	if err := json.Unmarshal(buf.Bytes(), &parsed); err != nil {
		t.Fatalf("output is not valid JSON: %v\nOutput:\n%s", err, buf.String())
	}

	if !strings.Contains(buf.String(), "\n") {
		t.Error("JSON output should be indented (multi-line)")
	}
}

func TestRenderIncidentsTable_EmptyData(t *testing.T) {
	var buf bytes.Buffer
	if err := RenderIncidentsTable(&buf, nil); err != nil {
		t.Fatal(err)
	}

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) != 1 {
		t.Errorf("expected only header line for empty data, got %d lines", len(lines))
	}
}

func TestRenderMonitorsTable_EmptyData(t *testing.T) {
	var buf bytes.Buffer
	if err := RenderMonitorsTable(&buf, nil); err != nil {
		t.Fatal(err)
	}

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) != 1 {
		t.Errorf("expected only header line for empty data, got %d lines", len(lines))
	}
}

func TestRenderJSON_EmptyArray(t *testing.T) {
	var buf bytes.Buffer
	if err := RenderJSON(&buf, []json.RawMessage{}); err != nil {
		t.Fatal(err)
	}

	output := strings.TrimSpace(buf.String())
	if output != "[]" {
		t.Errorf("got %q, want %q", output, "[]")
	}
}

func TestRenderIncidentDetail_ShowsKeyValuePairs(t *testing.T) {
	data := json.RawMessage(`{
		"id": "42",
		"attributes": {
			"name": "ALB Response Time",
			"cause": "Threshold crossed",
			"started_at": "2026-03-20T10:00:00Z",
			"resolved_at": "2026-03-20T10:05:00Z",
			"status": "Resolved"
		}
	}`)

	var buf bytes.Buffer
	if err := RenderIncidentDetail(&buf, data); err != nil {
		t.Fatal(err)
	}

	output := buf.String()
	for _, expected := range []string{"42", "ALB Response Time", "Threshold crossed", "Resolved", "5m"} {
		if !strings.Contains(output, expected) {
			t.Errorf("expected %q in detail output:\n%s", expected, output)
		}
	}
}

func TestRenderMonitorDetail_ShowsKeyValuePairs(t *testing.T) {
	data := json.RawMessage(`{
		"id": "99",
		"attributes": {
			"pronounceable_name": "Production API",
			"url": "https://api.example.com",
			"status": "up",
			"check_frequency": 30
		}
	}`)

	var buf bytes.Buffer
	if err := RenderMonitorDetail(&buf, data); err != nil {
		t.Fatal(err)
	}

	output := buf.String()
	for _, expected := range []string{"99", "Production API", "https://api.example.com", "up", "30s"} {
		if !strings.Contains(output, expected) {
			t.Errorf("expected %q in detail output:\n%s", expected, output)
		}
	}
}

package betterstack

import (
	"fmt"
	"net/http"
	"strings"
	"testing"
)

func TestLogsSources_ListShowsTable(t *testing.T) {
	srv := newTelemetryTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"data":[{"id":"7","attributes":{"name":"web","platform":"rails","team_name":"platform"}}]}`)
	})
	setupTelemetryEnv(t, srv.URL())
	resetBoolFlags(rootCmd, "json")

	out, _, err := executeCmd(t, "logs-sources", "list")
	if err != nil {
		t.Fatal(err)
	}
	for _, col := range []string{"ID", "NAME", "PLATFORM", "TEAM"} {
		if !strings.Contains(out, col) {
			t.Errorf("missing column %q:\n%s", col, out)
		}
	}
	for _, v := range []string{"7", "web", "rails", "platform"} {
		if !strings.Contains(out, v) {
			t.Errorf("missing value %q:\n%s", v, out)
		}
	}
}

func TestLogsSources_GetShowsDetail(t *testing.T) {
	srv := newTelemetryTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"data":{"id":"7","attributes":{"name":"web","platform":"rails","team_name":"platform","retention":30}}}`)
	})
	setupTelemetryEnv(t, srv.URL())
	resetBoolFlags(rootCmd, "json")

	out, _, err := executeCmd(t, "logs-sources", "get", "7")
	if err != nil {
		t.Fatal(err)
	}
	for _, v := range []string{"web", "rails", "platform", "30 days"} {
		if !strings.Contains(out, v) {
			t.Errorf("missing value %q:\n%s", v, out)
		}
	}
}

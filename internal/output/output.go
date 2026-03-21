package output

import (
	"encoding/json"
	"fmt"
	"io"
	"text/tabwriter"
	"time"
)

type Incident struct {
	ID         string `json:"id"`
	Attributes struct {
		Name       string  `json:"name"`
		URL        string  `json:"url"`
		Cause      string  `json:"cause"`
		StartedAt  string  `json:"started_at"`
		ResolvedAt *string `json:"resolved_at"`
		Status     string  `json:"status"`
	} `json:"attributes"`
}

type Monitor struct {
	ID         string `json:"id"`
	Attributes struct {
		PronounceableName string `json:"pronounceable_name"`
		URL               string `json:"url"`
		Status            string `json:"status"`
		CheckFrequency    int    `json:"check_frequency"`
	} `json:"attributes"`
}

func RenderIncidentsTable(w io.Writer, data []json.RawMessage) error {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "ID\tNAME\tCAUSE\tSTARTED AT\tRESOLVED AT\tLENGTH\tSTATUS")

	for _, raw := range data {
		var inc Incident
		if err := json.Unmarshal(raw, &inc); err != nil {
			fmt.Fprintf(tw, "-\t(parse error)\t-\t-\t-\t-\t-\n")
			continue
		}

		resolvedAt := dash
		length := dash
		if inc.Attributes.ResolvedAt != nil && *inc.Attributes.ResolvedAt != "" {
			resolvedAt = formatTime(*inc.Attributes.ResolvedAt)
			length = calculateLength(inc.Attributes.StartedAt, *inc.Attributes.ResolvedAt)
		}

		cause := inc.Attributes.Cause
		if cause == "" {
			cause = dash
		}

		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
			inc.ID,
			inc.Attributes.Name,
			cause,
			formatTime(inc.Attributes.StartedAt),
			resolvedAt,
			length,
			inc.Attributes.Status,
		)
	}

	return tw.Flush()
}

func RenderIncidentDetail(w io.Writer, data json.RawMessage) error {
	var inc Incident
	if err := json.Unmarshal(data, &inc); err != nil {
		return fmt.Errorf("failed to parse incident: %w", err)
	}

	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintf(tw, "ID:\t%s\n", inc.ID)
	fmt.Fprintf(tw, "Name:\t%s\n", inc.Attributes.Name)
	fmt.Fprintf(tw, "Status:\t%s\n", inc.Attributes.Status)
	fmt.Fprintf(tw, "Cause:\t%s\n", orDash(inc.Attributes.Cause))
	fmt.Fprintf(tw, "Started At:\t%s\n", formatTime(inc.Attributes.StartedAt))
	if inc.Attributes.ResolvedAt != nil && *inc.Attributes.ResolvedAt != "" {
		fmt.Fprintf(tw, "Resolved At:\t%s\n", formatTime(*inc.Attributes.ResolvedAt))
		fmt.Fprintf(tw, "Length:\t%s\n", calculateLength(inc.Attributes.StartedAt, *inc.Attributes.ResolvedAt))
	} else {
		fmt.Fprintf(tw, "Resolved At:\t%s\n", dash)
		fmt.Fprintf(tw, "Length:\t%s\n", dash)
	}
	return tw.Flush()
}

func RenderMonitorsTable(w io.Writer, data []json.RawMessage) error {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "ID\tNAME\tURL\tSTATUS\tCHECK FREQUENCY")

	for _, raw := range data {
		var mon Monitor
		if err := json.Unmarshal(raw, &mon); err != nil {
			fmt.Fprintf(tw, "-\t(parse error)\t-\t-\t-\n")
			continue
		}

		freq := dash
		if mon.Attributes.CheckFrequency > 0 {
			freq = fmt.Sprintf("%ds", mon.Attributes.CheckFrequency)
		}

		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\n",
			mon.ID,
			mon.Attributes.PronounceableName,
			mon.Attributes.URL,
			mon.Attributes.Status,
			freq,
		)
	}

	return tw.Flush()
}

func RenderMonitorDetail(w io.Writer, data json.RawMessage) error {
	var mon Monitor
	if err := json.Unmarshal(data, &mon); err != nil {
		return fmt.Errorf("failed to parse monitor: %w", err)
	}

	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintf(tw, "ID:\t%s\n", mon.ID)
	fmt.Fprintf(tw, "Name:\t%s\n", mon.Attributes.PronounceableName)
	fmt.Fprintf(tw, "URL:\t%s\n", mon.Attributes.URL)
	fmt.Fprintf(tw, "Status:\t%s\n", mon.Attributes.Status)
	freq := dash
	if mon.Attributes.CheckFrequency > 0 {
		freq = fmt.Sprintf("%ds", mon.Attributes.CheckFrequency)
	}
	fmt.Fprintf(tw, "Check Frequency:\t%s\n", freq)
	return tw.Flush()
}

func RenderJSON(w io.Writer, data any) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(data)
}

const dash = "-"

func orDash(s string) string {
	if s == "" {
		return dash
	}
	return s
}

func formatTime(t string) string {
	parsed, err := time.Parse(time.RFC3339, t)
	if err != nil {
		return t
	}
	return parsed.UTC().Format("2006-01-02 15:04:05 UTC")
}

func calculateLength(start, end string) string {
	s, err1 := time.Parse(time.RFC3339, start)
	e, err2 := time.Parse(time.RFC3339, end)
	if err1 != nil || err2 != nil {
		return dash
	}
	d := e.Sub(s)
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm", int(d.Minutes()))
	}
	return fmt.Sprintf("%dh%dm", int(d.Hours()), int(d.Minutes())%60)
}

package duration

import (
	"strings"
	"testing"

	"github.com/tommymorgan/betterstack-cli/internal/errs"
)

func TestParseSeconds_AcceptsGoStyleDurations(t *testing.T) {
	cases := map[string]int{
		"30s":    30,
		"5m":     300,
		"2h":     7200,
		"24h":    86400,
		"1h30m":  5400,
		"500ms":  0, // subseconds truncate to integer seconds
		"1.5s":   1,
	}
	for input, want := range cases {
		got, err := ParseSeconds(input)
		if err != nil {
			t.Errorf("ParseSeconds(%q) unexpected error: %v", input, err)
			continue
		}
		if got != want {
			t.Errorf("ParseSeconds(%q) = %d, want %d", input, got, want)
		}
	}
}

func TestParseSeconds_RejectsBareNumber(t *testing.T) {
	_, err := ParseSeconds("300")
	if err == nil {
		t.Fatal("expected error for bare number")
	}
	if errs.CodeOf(err) != errs.ExitUserInput {
		t.Errorf("exit code = %d, want %d", errs.CodeOf(err), errs.ExitUserInput)
	}
	if !strings.Contains(err.Error(), "24h") || !strings.Contains(err.Error(), "max unit") {
		t.Errorf("error should mention max unit guidance, got: %s", err.Error())
	}
}

func TestParseSeconds_RejectsZero(t *testing.T) {
	_, err := ParseSeconds("0s")
	if err == nil {
		t.Fatal("expected error for zero duration")
	}
	if !strings.Contains(err.Error(), "positive") {
		t.Errorf("error should say 'positive', got: %s", err.Error())
	}
}

func TestParseSeconds_RejectsNegative(t *testing.T) {
	_, err := ParseSeconds("-5m")
	if err == nil {
		t.Fatal("expected error for negative duration")
	}
	if !strings.Contains(err.Error(), "positive") {
		t.Errorf("error should say 'positive', got: %s", err.Error())
	}
}

func TestParseSeconds_RejectsDayUnit(t *testing.T) {
	// Go's time.ParseDuration doesn't accept "d" - should error
	_, err := ParseSeconds("1d")
	if err == nil {
		t.Fatal("expected error for day unit")
	}
	if !strings.Contains(err.Error(), "max unit") {
		t.Errorf("error should mention max unit guidance, got: %s", err.Error())
	}
}

func TestParseSeconds_RejectsEmpty(t *testing.T) {
	_, err := ParseSeconds("")
	if err == nil {
		t.Fatal("expected error for empty string")
	}
}

func TestHuman_FormatsCommonDurations(t *testing.T) {
	cases := map[int]string{
		30:    "30s",
		300:   "5m",
		7200:  "2h",
		86400: "24h",
		90:    "90s",
		0:     "0s",
	}
	for seconds, want := range cases {
		got := Human(seconds)
		if got != want {
			t.Errorf("Human(%d) = %q, want %q", seconds, got, want)
		}
	}
}

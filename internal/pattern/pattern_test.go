package pattern

import (
	"strings"
	"testing"
)

func TestWhereCondition_SimplePattern(t *testing.T) {
	got := WhereCondition("AADSTS7000215")
	want := `message CONTAINS "AADSTS7000215"`
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestWhereCondition_EscapesQuotes(t *testing.T) {
	got := WhereCondition(`he said "hi"`)
	want := `message CONTAINS "he said \"hi\""`
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestWhereCondition_EscapesBackslashes(t *testing.T) {
	got := WhereCondition(`a\b\c`)
	want := `message CONTAINS "a\\b\\c"`
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestWhereCondition_AdversarialInput(t *testing.T) {
	// Pattern with SQL-injection-like tokens, regex metacharacters, and escape chars
	input := `"; DROP *\.`
	got := WhereCondition(input)
	// The result must begin and end with un-nested double quotes around an escaped payload
	if !strings.HasPrefix(got, `message CONTAINS "`) {
		t.Errorf("missing prefix: %s", got)
	}
	if !strings.HasSuffix(got, `"`) {
		t.Errorf("missing suffix: %s", got)
	}
	// The raw double-quote at position 0 must be escaped
	if !strings.Contains(got, `\"`) {
		t.Errorf("raw quote not escaped: %s", got)
	}
	// The raw backslash must be escaped
	if !strings.Contains(got, `\\`) {
		t.Errorf("raw backslash not escaped: %s", got)
	}
}

func TestWhereCondition_EscapesControlChars(t *testing.T) {
	got := WhereCondition("line1\nline2\ttab")
	if !strings.Contains(got, `\n`) {
		t.Errorf("newline not escaped: %s", got)
	}
	if !strings.Contains(got, `\t`) {
		t.Errorf("tab not escaped: %s", got)
	}
}

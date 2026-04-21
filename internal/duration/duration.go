package duration

import (
	"strings"
	"time"

	"github.com/tommymorgan/betterstack-cli/internal/errs"
)

// ParseSeconds parses a Go-style duration string and returns the integer seconds.
// Enforces: units required, strictly positive, max unit is hour (no day/week).
func ParseSeconds(value string) (int, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return 0, errs.New(errs.ExitUserInput, "duration is empty; use a duration like 5m, 30s, or 2h (max unit: h; use 24h, not 1d)")
	}

	if endsWithDigit(trimmed) {
		return 0, errs.New(errs.ExitUserInput,
			"duration %q is missing a unit; use 5m, 30s, 2h, etc. (max unit: h; use 24h, not 1d)", value)
	}

	d, err := time.ParseDuration(trimmed)
	if err != nil {
		return 0, errs.New(errs.ExitUserInput,
			"duration %q is invalid: %s (max unit: h; use 24h, not 1d)", value, err.Error())
	}

	if d <= 0 {
		return 0, errs.New(errs.ExitUserInput,
			"duration %q must be strictly positive", value)
	}

	return int(d.Seconds()), nil
}

// Human formats an integer-second duration back to a human-readable string.
func Human(seconds int) string {
	if seconds <= 0 {
		return "0s"
	}
	d := time.Duration(seconds) * time.Second
	switch {
	case d%time.Hour == 0:
		return formatInt(int(d/time.Hour)) + "h"
	case d%time.Minute == 0:
		return formatInt(int(d/time.Minute)) + "m"
	default:
		return formatInt(int(d/time.Second)) + "s"
	}
}

func endsWithDigit(s string) bool {
	if len(s) == 0 {
		return false
	}
	c := s[len(s)-1]
	return c >= '0' && c <= '9'
}

func formatInt(n int) string {
	// Avoid strconv import churn; tiny helper is clearer here.
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}

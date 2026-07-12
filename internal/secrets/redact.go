package secrets

import (
	"os"
	"strings"
)

const Redacted = "REDACTED"

// TravelpayoutsToken returns TRAVELPAYOUTS_TOKEN, or AVIASALES_TOKEN as alias.
func TravelpayoutsToken(override string) string {
	if override != "" {
		return override
	}
	if t := os.Getenv("TRAVELPAYOUTS_TOKEN"); t != "" {
		return t
	}
	return os.Getenv("AVIASALES_TOKEN")
}

// Redact replaces token occurrences in s.
func Redact(s, token string) string {
	if token == "" {
		return s
	}
	out := strings.ReplaceAll(s, token, Redacted)
	// Also redact token query param values if partially logged.
	if idx := strings.Index(out, "token="); idx >= 0 {
		rest := out[idx+len("token="):]
		if end := strings.IndexAny(rest, "& \t\n"); end >= 0 {
			out = out[:idx+len("token=")] + Redacted + rest[end:]
		} else {
			out = out[:idx+len("token=")] + Redacted
		}
	}
	return out
}

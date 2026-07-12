package model

import (
	"sort"
	"strings"
)

// MergeAirlines returns sorted unique non-empty IATA codes.
func MergeAirlines(codes ...string) []string {
	seen := make(map[string]bool)
	var out []string
	for _, c := range codes {
		c = strings.ToUpper(strings.TrimSpace(c))
		if c == "" || seen[c] {
			continue
		}
		seen[c] = true
		out = append(out, c)
	}
	sort.Strings(out)
	return out
}

// MergeAirlineLists merges multiple airline slices.
func MergeAirlineLists(lists ...[]string) []string {
	var flat []string
	for _, l := range lists {
		flat = append(flat, l...)
	}
	return MergeAirlines(flat...)
}

// FormatAirlineList joins codes for display; empty slice → "n/a".
func FormatAirlineList(codes []string) string {
	if len(codes) == 0 {
		return "n/a"
	}
	return strings.Join(codes, ", ")
}

package links_test

import (
	"testing"

	"github.com/kneumoin/nepal/internal/links"
)

func TestOneWaySearchURL(t *testing.T) {
	got, err := links.OneWaySearchURL("MOW", "KTM", "2026-09-28", 1)
	if err != nil {
		t.Fatal(err)
	}
	want := "https://www.aviasales.ru/search/MOW2809KTM1"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestRoundTripShortURL(t *testing.T) {
	got, err := links.RoundTripShortURL("MOW", "KTM", "2026-09-26", "2026-11-13", 1)
	if err != nil {
		t.Fatal(err)
	}
	want := "https://www.aviasales.ru/search/MOW2609KTM13111"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestRoundTripQueryURL(t *testing.T) {
	got := links.RoundTripQueryURL("MOW", "KTM", "2026-09-26", "2026-11-13", 1)
	if got == "" || !containsAll(got, "origin_iata=MOW", "destination_iata=KTM", "depart_date=2026-09-26", "return_date=2026-11-13") {
		t.Fatalf("unexpected url: %q", got)
	}
}

func containsAll(s string, parts ...string) bool {
	for _, p := range parts {
		if !contains(s, p) {
			return false
		}
	}
	return true
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(sub) == 0 || indexOf(s, sub) >= 0)
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}

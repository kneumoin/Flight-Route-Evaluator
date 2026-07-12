package config

import "testing"

func TestOutboundFirstLegDates(t *testing.T) {
	cfg := &Config{
		Trip:     TripConfig{DepartureDate: "2026-09-26"},
		Coverage: CoverageConfig{OutboundForwardDays: 6},
	}
	dates := cfg.OutboundFirstLegDates()
	if len(dates) != 7 {
		t.Fatalf("expected 7 dates, got %d: %v", len(dates), dates)
	}
	if dates[0] != "2026-09-26" || dates[len(dates)-1] != "2026-10-02" {
		t.Fatalf("unexpected range: %v", dates)
	}
}

func TestReturnFirstLegDates(t *testing.T) {
	cfg := &Config{
		Trip: TripConfig{ReturnDate: "2026-11-11", ReturnDateEnd: "2026-11-15"},
	}
	dates := cfg.ReturnFirstLegDates()
	if len(dates) != 5 {
		t.Fatalf("expected 5 dates, got %d: %v", len(dates), dates)
	}
	if dates[0] != "2026-11-11" || dates[4] != "2026-11-15" {
		t.Fatalf("unexpected range: %v", dates)
	}
}

func TestValidate_ReturnDateRange(t *testing.T) {
	cfg := &Config{
		Trip: TripConfig{
			Origin: "MOW", Destination: "KTM",
			DepartureDate: "2026-09-26",
			ReturnDate:    "2026-11-15",
			ReturnDateEnd: "2026-11-11",
			Passengers:    1, Cabin: "economy",
		},
		Scoring: ScoringConfig{Currency: "USD"},
		Branches: []BranchConfig{{
			ID: "via_doh", Name: "Via Doha", Type: "mixed_carrier", VisaPolicy: "airside_only",
			MinConnectionHours: 3, MaxConnectionHours: 12,
			Legs: []LegConfig{{From: "MOW", To: "DOH"}, {From: "DOH", To: "KTM"}},
		}},
	}
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected error for reversed return range")
	}
}

func TestRoundTripEnabled(t *testing.T) {
	cfg := &Config{Trip: TripConfig{ReturnDate: "2026-11-11"}}
	if !cfg.RoundTrip() {
		t.Fatal("expected round trip")
	}
	cfg.Trip.ReturnDate = ""
	if cfg.RoundTrip() {
		t.Fatal("expected one-way")
	}
}

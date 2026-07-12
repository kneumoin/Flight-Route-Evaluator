package config

import (
	"path/filepath"
	"testing"
)

func TestLoad_ConfigsRoutes(t *testing.T) {
	cfg, err := Load(filepath.Join("..", "..", "configs", "routes.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.Branches) < 10 {
		t.Fatalf("branches=%d, want >= 10", len(cfg.Branches))
	}
}

func TestCacheConfig_TTLDuration(t *testing.T) {
	c := CacheConfig{TTL: "6h"}
	d, err := c.TTLDuration()
	if err != nil || d.Hours() != 6 {
		t.Fatalf("got %v err=%v", d, err)
	}
}

func TestValidate_EmptyBranchesOK(t *testing.T) {
	cfg := &Config{
		Trip: TripConfig{Origin: "MOW", Destination: "KTM", DepartureDate: "2026-09-28", Passengers: 1, Cabin: "economy"},
		Scoring: ScoringConfig{Currency: "USD"},
		Branches: nil,
	}
	if err := cfg.Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestValidate_InvalidDate(t *testing.T) {
	cfg := &Config{
		Trip: TripConfig{Origin: "MOW", Destination: "KTM", DepartureDate: "bad", Passengers: 1},
		Scoring: ScoringConfig{Currency: "USD"},
	}
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected error")
	}
}

func TestValidate_NegativeConnectionHours(t *testing.T) {
	cfg := &Config{
		Trip: TripConfig{Origin: "MOW", Destination: "KTM", DepartureDate: "2026-09-28", Passengers: 1, Cabin: "economy"},
		Scoring: ScoringConfig{Currency: "USD"},
		Branches: []BranchConfig{{
			ID: "x", Name: "x", Type: "single_ticket", VisaPolicy: "airside_only",
			MinConnectionHours: -1, MaxConnectionHours: 5,
			Legs: []LegConfig{{From: "MOW", To: "DOH"}},
		}},
	}
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected error")
	}
}

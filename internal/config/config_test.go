package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoad_Valid(t *testing.T) {
	cfg, err := Load(filepath.Join("..", "..", "configs", "routes.yaml"))
	if err != nil {
		// config may not exist yet during early test runs
		if os.IsNotExist(err) {
			t.Skip("configs/routes.yaml not yet created")
		}
		t.Fatal(err)
	}
	if len(cfg.Branches) != 5 {
		t.Fatalf("expected 5 branches, got %d", len(cfg.Branches))
	}
}

func TestValidate_InvalidIATA(t *testing.T) {
	cfg := &Config{
		Trip: TripConfig{Origin: "XX", Destination: "KTM", DepartureDate: "2026-09-28", Passengers: 1, Cabin: "economy"},
		Scoring: ScoringConfig{Currency: "USD"},
	}
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected error")
	}
}

func TestValidate_UnknownVisaPolicy(t *testing.T) {
	cfg := &Config{
		Trip: TripConfig{Origin: "MOW", Destination: "KTM", DepartureDate: "2026-09-28", Passengers: 1, Cabin: "economy"},
		Scoring: ScoringConfig{Currency: "USD"},
		Branches: []BranchConfig{{
			ID: "x", Name: "x", Type: "single_ticket", VisaPolicy: "bad",
			MinConnectionHours: 1, MaxConnectionHours: 5,
			Legs: []LegConfig{{From: "MOW", To: "DOH"}},
		}},
	}
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected error")
	}
}

func TestValidate_MinMaxConnection(t *testing.T) {
	cfg := &Config{
		Trip: TripConfig{Origin: "MOW", Destination: "KTM", DepartureDate: "2026-09-28", Passengers: 1, Cabin: "economy"},
		Scoring: ScoringConfig{Currency: "USD"},
		Branches: []BranchConfig{{
			ID: "x", Name: "x", Type: "single_ticket", VisaPolicy: "airside_only",
			MinConnectionHours: 10, MaxConnectionHours: 5,
			Legs: []LegConfig{{From: "MOW", To: "DOH"}},
		}},
	}
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected error")
	}
}

func TestLoad_FixtureInvalid(t *testing.T) {
	path := filepath.Join("..", "..", "testdata", "config", "invalid_iata.yaml")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Skip("fixture not yet created")
	}
	_, err := Load(path)
	if err == nil {
		t.Fatal("expected error")
	}
}

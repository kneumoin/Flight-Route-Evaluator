package scoring_test

import (
	"testing"

	"github.com/kneumoin/nepal/internal/config"
	"github.com/kneumoin/nepal/internal/scoring"
)

func TestWorstVisaFromLegs_TashkentDelhi(t *testing.T) {
	legs := []config.LegConfig{
		{From: "MOW", To: "TAS"},
		{From: "TAS", To: "DEL"},
		{From: "DEL", To: "KTM"},
	}
	got := scoring.WorstVisaFromLegs(legs)
	if got != scoring.VisaRequiresVisa {
		t.Fatalf("visa=%s want REQUIRES_VISA (Delhi)", got)
	}
}

func TestIntermediateHubsFromLegs(t *testing.T) {
	legs := []config.LegConfig{
		{From: "MOW", To: "DYU"},
		{From: "DYU", To: "DEL"},
		{From: "DEL", To: "KTM"},
	}
	got := scoring.IntermediateHubsFromLegs(legs)
	if len(got) != 2 || got[0] != "DYU" || got[1] != "DEL" {
		t.Fatalf("hubs=%v", got)
	}
}

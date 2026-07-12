package config_test

import (
	"path/filepath"
	"testing"

	"github.com/kneumoin/nepal/internal/config"
	"github.com/kneumoin/nepal/internal/scoring"
)

func TestRoutesConfig_ApprovedHubs(t *testing.T) {
	cfg, err := config.Load(filepath.Join("..", "..", "configs", "routes.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	oneStop := 0
	multiStop := 0
	seen := map[string]bool{}
	for _, b := range cfg.Branches {
		switch len(b.Legs) {
		case 2:
			oneStop++
			if b.Legs[0].From != "MOW" || b.Legs[1].To != "KTM" {
				t.Fatalf("%s: bad leg pattern", b.ID)
			}
			hub := b.Legs[0].To
			if b.Legs[1].From != hub {
				t.Fatalf("%s: hub mismatch", b.ID)
			}
			seen[hub] = true
		case 3:
			multiStop++
			if b.Legs[0].From != "MOW" || b.Legs[2].To != "KTM" {
				t.Fatalf("%s: bad multi-stop pattern", b.ID)
			}
		default:
			t.Fatalf("%s: want 2 or 3 legs", b.ID)
		}
		if b.Legs[0].ProviderHint != "" {
			t.Fatalf("%s: provider_hint present", b.ID)
		}
	}
	// The active config is a curated subset (Middle East + Central Asia + India
	// gateways). Every configured hub must still be on the approved whitelist,
	// but we no longer require a branch for every approved hub.
	if oneStop == 0 {
		t.Fatalf("expected at least one one-stop branch")
	}
	if oneStop > len(scoring.ApprovedTransitHubs) {
		t.Fatalf("one-stop branches=%d exceeds approved hubs=%d", oneStop, len(scoring.ApprovedTransitHubs))
	}
	if multiStop < 2 {
		t.Fatalf("expected at least 2 two-transfer branches, got %d", multiStop)
	}
	approved := map[string]bool{}
	for _, h := range scoring.ApprovedTransitHubs {
		approved[h] = true
	}
	for hub := range seen {
		if !approved[hub] {
			t.Fatalf("configured hub %s is not on the approved whitelist", hub)
		}
	}
}

func TestValidHubIATA_HKG(t *testing.T) {
	if !config.IsValidHubIATA("HKG") {
		t.Fatal("HKG should be valid")
	}
}

func TestValidHubIATA_HGK_Invalid(t *testing.T) {
	if config.IsValidHubIATA("HGK") {
		t.Fatal("HGK should be invalid typo")
	}
}

func TestHubVisaRuleExists(t *testing.T) {
	for _, h := range scoring.ApprovedTransitHubs {
		cat := scoring.HubVisaCategory(h)
		if cat == "" {
			t.Fatalf("no visa rule for %s", h)
		}
	}
}

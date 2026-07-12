package search_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/kneumoin/nepal/internal/config"
	"github.com/kneumoin/nepal/internal/model"
	"github.com/kneumoin/nepal/internal/provider"
	"github.com/kneumoin/nepal/internal/provider/mock"
	"github.com/kneumoin/nepal/internal/scoring"
	"github.com/kneumoin/nepal/internal/search"
)

func TestEvaluate_GulfOperationalDisruptionNotRejected(t *testing.T) {
	cfg, err := config.Load(filepath.Join("..", "..", "configs", "routes.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	cfg.Providers = []config.ProviderConfig{{ID: "mock", Enabled: true}}
	reg := provider.NewRegistry()
	reg.Register(mock.New())
	res, err := (&search.Evaluator{Config: cfg, Registry: reg, MockOnly: true}).Evaluate(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	for _, br := range res.Branches {
		if br.BranchID != "via_doh" {
			continue
		}
		if br.OperationalDisruptionRisk != model.OperationalDisruptionHigh {
			t.Fatalf("DOH risk=%s", br.OperationalDisruptionRisk)
		}
		if br.Status == model.StatusRejected {
			t.Fatal("Gulf hub should not be rejected for disruption risk")
		}
		return
	}
	t.Fatal("via_doh not found")
}

func TestEvaluate_ISTOperationalDisruptionLow(t *testing.T) {
	cfg, err := config.Load(filepath.Join("..", "..", "configs", "routes.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	cfg.Providers = []config.ProviderConfig{{ID: "mock", Enabled: true}}
	reg := provider.NewRegistry()
	reg.Register(mock.New())
	res, err := (&search.Evaluator{Config: cfg, Registry: reg, MockOnly: true}).Evaluate(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	for _, br := range res.Branches {
		if br.BranchID == "via_ist" && br.Status == model.StatusOK {
			if br.OperationalDisruptionRisk != model.OperationalDisruptionLow {
				t.Fatalf("IST risk=%s", br.OperationalDisruptionRisk)
			}
			if br.OperationalDisruptionPenalty != 0 {
				t.Fatal("IST should have no disruption penalty")
			}
			return
		}
	}
}

func TestEvaluate_UnlistedHubDefaultsLow(t *testing.T) {
	if scoring.HubOperationalDisruptionRisk("DEL", config.OperationalDisruptionConfig{
		Enabled: true, DefaultLevel: "LOW",
		Hubs: map[string]string{"DOH": "HIGH"},
	}) != model.OperationalDisruptionLow {
		t.Fatal("DEL should default LOW")
	}
}

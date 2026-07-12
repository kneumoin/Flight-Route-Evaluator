package search_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/kneumoin/nepal/internal/config"
	"github.com/kneumoin/nepal/internal/model"
	"github.com/kneumoin/nepal/internal/provider"
	"github.com/kneumoin/nepal/internal/provider/mock"
	"github.com/kneumoin/nepal/internal/search"
)

func TestEvaluate_PriceComparisonOnOKBranch(t *testing.T) {
	cfg, err := config.Load(filepath.Join("..", "..", "testdata", "routes.yaml"))
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
	var doh *model.BranchResult
	for i := range res.Branches {
		if res.Branches[i].BranchID == "qatar_doha" && res.Branches[i].Status == model.StatusOK {
			doh = &res.Branches[i]
			break
		}
	}
	if doh == nil {
		t.Fatal("qatar_doha should be OK with mock data")
	}
	if doh.PriceComparison == nil || doh.PriceComparison.PriceTarget == nil {
		t.Fatal("expected price comparison on OK branch")
	}
	if doh.PriceComparison.PriceWindowDays != 14 {
		t.Fatalf("window days=%d", doh.PriceComparison.PriceWindowDays)
	}
}

func TestEvaluate_DelhiVisaCategoryRequiresVisa(t *testing.T) {
	cfg, err := config.Load(filepath.Join("..", "..", "testdata", "routes.yaml"))
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
		if br.BranchID != "delhi_mixed" {
			continue
		}
		if br.VisaCategory != model.VisaCategoryRequiresVisa {
			t.Fatalf("DEL visa category=%s", br.VisaCategory)
		}
		if br.Status == model.StatusRejected {
			t.Fatal("DEL should not be rejected for visa alone")
		}
		return
	}
	t.Fatal("delhi_mixed not found")
}

func TestEvaluate_LegAirlinesOnOKBranch(t *testing.T) {
	cfg, err := config.Load(filepath.Join("..", "..", "testdata", "routes.yaml"))
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
		if br.BranchID != "qatar_doha" || br.Status != model.StatusOK {
			continue
		}
		if len(br.LegAirlines) != 2 {
			t.Fatalf("leg_airlines=%d", len(br.LegAirlines))
		}
		if len(br.LegAirlines[0].AvailableAirlines) == 0 {
			t.Fatal("expected airlines on leg1")
		}
		return
	}
	t.Fatal("qatar_doha not ok")
}

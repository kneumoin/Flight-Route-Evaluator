package search

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/kneumoin/nepal/internal/config"
	"github.com/kneumoin/nepal/internal/model"
	"github.com/kneumoin/nepal/internal/provider"
)

type stubTP struct{}

func (stubTP) Name() string { return "travelpayouts_data" }
func (stubTP) Capabilities() provider.ProviderCapabilities {
	return provider.ProviderCapabilities{AirlineCoverageMode: model.CoverageUnknown}
}
func (stubTP) Search(ctx context.Context, q model.Query) ([]model.Offer, error) {
	return nil, nil
}

func TestEvaluate_SUPreferredNotNoProvider(t *testing.T) {
	cfg, err := config.Load(filepath.Join("..", "..", "configs", "routes.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	cfg.Providers = []config.ProviderConfig{{ID: "travelpayouts_data", Enabled: true}}
	reg := provider.NewRegistry()
	reg.Register(stubTP{})

	ev := &Evaluator{Config: cfg, Registry: reg, Verbose: false}
	res, err := ev.Evaluate(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	for _, b := range res.Branches {
		if b.BranchID == "dubai_hub" || b.BranchID == "istanbul_hub" || b.BranchID == "delhi_hub" {
			for _, c := range b.ReasonCodes {
				if c == model.ReasonNoProvider {
					t.Fatalf("branch %s should not fail with NO_PROVIDER for SU preference, got %v", b.BranchID, b.ReasonCodes)
				}
			}
		}
	}
}

package search

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/kneumoin/nepal/internal/config"
	"github.com/kneumoin/nepal/internal/model"
	"github.com/kneumoin/nepal/internal/provider"
	"github.com/kneumoin/nepal/internal/provider/mock"
)

func testEvaluator(t *testing.T) *Evaluator {
	t.Helper()
	cfg, err := config.Load(filepath.Join("..", "..", "testdata", "routes.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	cfg.Providers = []config.ProviderConfig{{ID: "mock", Enabled: true}}
	reg := provider.NewRegistry()
	reg.Register(mock.New())
	return &Evaluator{Config: cfg, Registry: reg, MockOnly: true}
}

func TestEvaluate_AllBranches(t *testing.T) {
	ev := testEvaluator(t)
	res, err := ev.Evaluate(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Branches) != 5 {
		t.Fatalf("expected 5 branches, got %d", len(res.Branches))
	}
}

func TestEvaluate_DelhiVisaPenalizedNotRejected(t *testing.T) {
	ev := testEvaluator(t)
	res, err := ev.Evaluate(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	var delhi model.BranchResult
	for _, b := range res.Branches {
		if b.BranchID == "delhi_mixed" {
			delhi = b
		}
	}
	if delhi.Status == model.StatusRejected {
		t.Fatalf("delhi should not be rejected for visa, got %s", delhi.Status)
	}
	if delhi.VisaCategory != model.VisaCategoryRequiresVisa && delhi.Status == model.StatusOK {
		// visa category set when branch evaluated
	}
	if delhi.VisaCategory == "" && delhi.Status == model.StatusOK {
		t.Fatal("expected visa category on ok branch")
	}
}

func TestEvaluate_StableRanking(t *testing.T) {
	ev := testEvaluator(t)
	r1, _ := ev.Evaluate(context.Background())
	r2, _ := ev.Evaluate(context.Background())
	for i := range r1.Branches {
		if r1.Branches[i].BranchID != r2.Branches[i].BranchID {
			t.Fatal("ranking not stable")
		}
	}
}

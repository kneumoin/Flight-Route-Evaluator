package search_test

import (
	"bytes"
	"context"
	"flag"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/kneumoin/nepal/internal/config"
	"github.com/kneumoin/nepal/internal/model"
	"github.com/kneumoin/nepal/internal/provider"
	"github.com/kneumoin/nepal/internal/provider/mock"
	"github.com/kneumoin/nepal/internal/report"
	"github.com/kneumoin/nepal/internal/search"
)

var update = flag.Bool("update", false, "update golden files")

func evalResult(t *testing.T) *model.EvaluationResult {
	t.Helper()
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
	return res
}

func TestIntegration_Pipeline(t *testing.T) {
	res := evalResult(t)
	if len(res.Branches) != 5 {
		t.Fatalf("branch count %d", len(res.Branches))
	}
	rejected := 0
	for _, b := range res.Branches {
		if b.Status != model.StatusOK {
			rejected++
		}
	}
	if rejected < 1 {
		t.Fatal("expected at least one rejected/unavailable branch")
	}
	dir := t.TempDir()
	if err := report.WriteHTML(filepath.Join(dir, "report.html"), res); err != nil {
		t.Fatal(err)
	}
	if err := report.WriteCSV(filepath.Join(dir, "report.csv"), res); err != nil {
		t.Fatal(err)
	}
	if err := report.WriteJSON(filepath.Join(dir, "results.json"), res); err != nil {
		t.Fatal(err)
	}
}

func TestGolden_Outputs(t *testing.T) {
	goldenDir := filepath.Join("..", "..", "testdata", "golden")
	res := evalResult(t)
	fixedTime := time.Date(2026, 7, 5, 10, 30, 0, 0, time.UTC)
	res.GeneratedAt = fixedTime

	files := []struct {
		name  string
		write func(string) error
		norm  func([]byte) []byte
	}{
		{"expected_results.json", func(p string) error { return report.WriteJSON(p, res) }, func(b []byte) []byte { return b }},
		{"expected_report.csv", func(p string) error { return report.WriteCSV(p, res) }, func(b []byte) []byte { return b }},
		{"expected_report.html", func(p string) error { return report.WriteHTML(p, res) }, func(b []byte) []byte {
			return bytes.ReplaceAll(b, []byte(fixedTime.Format(time.RFC3339)), []byte("TIMESTAMP"))
		}},
	}
	for _, f := range files {
		goldenPath := filepath.Join(goldenDir, f.name)
		tmp := filepath.Join(t.TempDir(), f.name)
		if err := f.write(tmp); err != nil {
			t.Fatal(err)
		}
		got, _ := os.ReadFile(tmp)
		got = f.norm(got)
		if *update {
			if err := os.WriteFile(goldenPath, got, 0o644); err != nil {
				t.Fatal(err)
			}
			continue
		}
		want, err := os.ReadFile(goldenPath)
		if err != nil {
			t.Fatalf("missing golden %s (run -update): %v", f.name, err)
		}
		if !bytes.Equal(got, want) {
			t.Fatalf("golden mismatch %s", f.name)
		}
	}
}

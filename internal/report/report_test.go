package report

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/kneumoin/nepal/internal/model"
)

func sampleResult() *model.EvaluationResult {
	score := 85.0
	bag := 30
	return &model.EvaluationResult{
		GeneratedAt: time.Date(2026, 7, 5, 10, 30, 0, 0, time.UTC),
		Trip:        model.TripMeta{Origin: "MOW", Destination: "KTM", DepartureDate: "2026-09-28", Passengers: 1, Cabin: "economy"},
		Branches: []model.BranchResult{{
			BranchID: "qatar_doha", BranchName: "Qatar via Doha", Status: model.StatusOK, Score: &score,
			Offer: &model.Offer{
				Price: model.Money{Amount: 85000, Currency: "USD"}, VisaRisk: model.RiskLow,
				CheckedBaggageKg: &bag, TotalDuration: 18 * time.Hour,
			},
		}},
	}
}

func TestHTML_ContainsBilingualAndOffline(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "report.html")
	if err := WriteHTML(path, sampleResult()); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	s := string(b)
	for _, want := range []string{"Сводный рейтинг", "Summary Ranking", "setLang('ru')", "setLang('en')", "eval-data"} {
		if !strings.Contains(s, want) {
			t.Fatalf("missing %q", want)
		}
	}
	if strings.Contains(s, "cdn.jsdelivr") || strings.Contains(s, "unpkg.com") {
		t.Fatal("external CDN found")
	}
}

func TestCSV_WritesHeaders(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "report.csv")
	if err := WriteCSV(path, sampleResult()); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(path)
	if !strings.Contains(string(b), "price_normalized") {
		t.Fatal("missing price_normalized column")
	}
}

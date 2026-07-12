package report_test

import (
	"strings"
	"testing"
	"time"

	"github.com/kneumoin/nepal/internal/model"
	"github.com/kneumoin/nepal/internal/report"
)

func TestHTML_VisaBoldWarning(t *testing.T) {
	score := 50.0
	res := &model.EvaluationResult{
		GeneratedAt: time.Now(),
		Trip:        model.TripMeta{Origin: "MOW", Destination: "KTM", DepartureDate: "2026-09-28"},
		Branches: []model.BranchResult{{
			BranchID: "via_del", BranchName: "Via Delhi", Status: model.StatusOK,
			VisaCategory: model.VisaCategoryRequiresVisa,
			Score:        &score,
			PriceComparison: &model.PriceComparison{
				PriceTarget: &model.Money{Amount: 100000, Currency: "USD"},
				PriceWindowDays: 14,
			},
		}},
	}
	html := report.RenderHTMLForTest(res)
	if !strings.Contains(html, "visa-warn-bold") {
		t.Fatal("expected bold visa warning class")
	}
	if !strings.Contains(html, `data-i18n="visa_required"`) {
		t.Fatal("expected visa_required i18n key")
	}
}

func TestHTML_BranchAirlines(t *testing.T) {
	res := &model.EvaluationResult{
		GeneratedAt: time.Now(),
		Trip:        model.TripMeta{Origin: "MOW", Destination: "KTM", DepartureDate: "2026-09-28"},
		Branches: []model.BranchResult{{
			BranchID: "via_dxb", BranchName: "Via Dubai", Status: model.StatusOK,
			LegAirlines: []model.LegAirlines{
				{From: "MOW", To: "DXB", TargetDate: "2026-09-28", AvailableAirlines: []string{"DP", "SU"}},
				{From: "DXB", To: "KTM", TargetDate: "2026-09-28", AvailableAirlines: []string{"QR", "FZ"}},
			},
		}},
	}
	html := report.RenderHTMLForTest(res)
	if !strings.Contains(html, "DP, SU") || !strings.Contains(html, "QR, FZ") {
		t.Fatal("expected branch airlines in html")
	}
}

func TestHTML_OperationalDisruptionBoldWarning(t *testing.T) {
	res := &model.EvaluationResult{
		GeneratedAt: time.Now(),
		Trip:        model.TripMeta{Origin: "MOW", Destination: "KTM", DepartureDate: "2026-09-28"},
		Branches: []model.BranchResult{{
			BranchID: "via_doh", BranchName: "Via Doha", Status: model.StatusOK,
			OperationalDisruptionRisk:    model.OperationalDisruptionHigh,
			OperationalDisruptionPenalty: 12,
			OperationalDisruptionNotes:   model.OperationalDisruptionNotes(model.OperationalDisruptionHigh),
		}},
	}
	html := report.RenderHTMLForTest(res)
	if !strings.Contains(html, "regional-warn-bold") {
		t.Fatal("expected bold disruption warning")
	}
	if !strings.Contains(html, `data-i18n="ops_high"`) {
		t.Fatal("expected ops_high i18n key")
	}
}

func TestPriceComparison_CompactFormat(t *testing.T) {
	pc := &model.PriceComparison{
		PriceTarget:     &model.Money{Amount: 100000, Currency: "USD"},
		PriceMinus1:     &model.Money{Amount: 108000, Currency: "USD"},
		PricePlus1:      &model.Money{Amount: 99000, Currency: "USD"},
		PriceWindowMin:  &model.Money{Amount: 95000, Currency: "USD"},
		PriceWindowDays: 14,
	}
	s := pc.FormatPriceComparisonCompact("en")
	if !strings.Contains(s, "$1000") || !strings.Contains(s, "14d min") {
		t.Fatalf("compact=%q", s)
	}
	if strings.Contains(s, "n/a") && pc.PriceMinus1 != nil {
		t.Fatalf("unexpected n/a: %s", s)
	}
}

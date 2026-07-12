package search

import (
	"testing"

	"github.com/kneumoin/nepal/internal/model"
)

func TestCoverageRowHasInfo(t *testing.T) {
	if coverageRowHasInfo(model.RouteCoverageRow{}) {
		t.Fatal("empty row should not have info")
	}
	if !coverageRowHasInfo(model.RouteCoverageRow{CoverageDays: 2}) {
		t.Fatal("coverage days should count")
	}
	if !coverageRowHasInfo(model.RouteCoverageRow{AvailableAirlines: []string{"FZ"}}) {
		t.Fatal("airlines should count")
	}
}

func TestOffersFromCoverage_EstimatedDate(t *testing.T) {
	row := model.RouteCoverageRow{
		CheapestDateNearTarget: "2026-09-17",
		MinPrice:               &model.Money{Amount: 16900, Currency: "USD"},
		Airline:                "AI",
		AvailableAirlines:      []string{"AI", "FZ"},
		CoverageDays:           5,
	}
	offers := offersFromCoverage(row, "DXB", "KTM", "2026-09-28", "USD")
	if len(offers) != 1 {
		t.Fatalf("offers=%d", len(offers))
	}
	o := offers[0]
	if !o.EstimatedDate || o.SearchDate != "2026-09-17" {
		t.Fatalf("expected estimated nearby date, got search=%s estimated=%v", o.SearchDate, o.EstimatedDate)
	}
	if o.Price.Amount != 16900 {
		t.Fatalf("price=%d", o.Price.Amount)
	}
	if len(o.AvailableAirlines) < 2 {
		t.Fatalf("airlines=%v", o.AvailableAirlines)
	}
}

func TestOffersFromCoverage_NoPrice(t *testing.T) {
	row := model.RouteCoverageRow{
		AvailableAirlines: []string{"EK", "FZ"},
		CoverageDays:      3,
	}
	offers := offersFromCoverage(row, "MOW", "DXB", "2026-09-28", "USD")
	if len(offers) != 1 {
		t.Fatal("expected offer shell without price")
	}
	if offers[0].Price.Amount != 0 {
		t.Fatalf("price=%d", offers[0].Price.Amount)
	}
}

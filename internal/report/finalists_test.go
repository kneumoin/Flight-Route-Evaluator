package report

import (
	"strings"
	"testing"
	"time"

	"github.com/kneumoin/nepal/internal/model"
)

func finalistsSampleResult() *model.EvaluationResult {
	score := func(v float64) *float64 { return &v }
	return &model.EvaluationResult{
		GeneratedAt: time.Date(2026, 7, 12, 13, 0, 0, 0, time.UTC),
		Trip: model.TripMeta{
			Origin: "MOW", Destination: "KTM",
			DepartureDate: "2026-09-26", ReturnDate: "2026-11-11", ReturnDateEnd: "2026-11-15",
			Passengers: 1, Cabin: "economy",
		},
		Branches: []model.BranchResult{
			{
				BranchID: "via_doh", BranchName: "Via Doha", Status: model.StatusOK, Score: score(90),
				OutboundOffer: &model.Offer{
					Price: model.Money{Amount: 50000, Currency: "USD"},
					LegDetails: []model.LegDetail{
						{From: "MOW", To: "DOH", SearchDate: "2026-09-26", Airline: "QR"},
						{From: "DOH", To: "KTM", SearchDate: "2026-09-27", Airline: "QR"},
					},
				},
				ReturnOffer: &model.Offer{
					Price: model.Money{Amount: 35000, Currency: "USD"},
					LegDetails: []model.LegDetail{
						{From: "KTM", To: "DOH", SearchDate: "2026-11-11", Airline: "QR"},
						{From: "DOH", To: "MOW", SearchDate: "2026-11-12", Airline: "QR"},
					},
				},
			},
			{
				BranchID: "via_ist", BranchName: "Via Istanbul", Status: model.StatusPartial, Score: score(70),
				Offer: &model.Offer{
					Price:      model.Money{Amount: 120000, Currency: "USD"},
					LegDetails: []model.LegDetail{{From: "MOW", To: "IST", SearchDate: "2026-09-26", Airline: "TK"}},
				},
			},
			{
				BranchID: "via_nope", BranchName: "Rejected", Status: model.StatusUnavailable, Score: nil,
			},
		},
	}
}

func TestFinalistsHTMLTopN(t *testing.T) {
	html := RenderFinalistsHTMLForTest(finalistsSampleResult(), 3)

	for _, want := range []string{
		"Via Doha", "MOW → DOH → KTM", "≈ $850",
		"Туда: 26.09.2026", "Обратно: 11.11.2026",
		"aviasales.ru/search/MOW2609KTM11111",
		"Via Istanbul",
	} {
		if !strings.Contains(html, want) {
			t.Errorf("finalists html missing %q", want)
		}
	}
	if strings.Contains(html, "Rejected") {
		t.Errorf("unavailable branch should not appear in finalists")
	}
}

func TestFinalistsHTMLLimit(t *testing.T) {
	html := RenderFinalistsHTMLForTest(finalistsSampleResult(), 1)
	if !strings.Contains(html, "Via Doha") {
		t.Fatalf("top-1 must include best branch")
	}
	if strings.Contains(html, "Via Istanbul") {
		t.Errorf("top-1 must not include second branch")
	}
}

package report_test

import (
	"strings"
	"testing"
	"time"

	"github.com/kneumoin/nepal/internal/model"
	"github.com/kneumoin/nepal/internal/report"
)

func TestBookingHTML_ContainsSnapshotAndLink(t *testing.T) {
	price := model.Money{Amount: 84000, Currency: "USD"}
	result := &model.EvaluationResult{
		GeneratedAt: time.Date(2026, 7, 12, 12, 45, 0, 0, time.UTC),
		Trip: model.TripMeta{
			Origin: "MOW", Destination: "KTM",
			DepartureDate: "2026-09-26",
			ReturnDate:    "2026-11-11",
			Passengers:    1,
		},
		Branches: []model.BranchResult{{
			BranchID: "via_kul", BranchName: "Via Kuala Lumpur", Status: model.StatusOK,
			OutboundOffer: &model.Offer{
				Price: price,
				LegDetails: []model.LegDetail{{
					From: "MOW", To: "KUL", SearchDate: "2026-09-26",
					Airline: "QR", Price: model.Money{Amount: 40000, Currency: "USD"},
				}, {
					From: "KUL", To: "KTM", SearchDate: "2026-09-27",
					Airline: "MH", Price: model.Money{Amount: 44000, Currency: "USD"},
				}},
			},
			ReturnOffer: &model.Offer{
				Price: model.Money{Amount: 70000, Currency: "USD"},
				LegDetails: []model.LegDetail{{
					From: "KTM", To: "KUL", SearchDate: "2026-11-13",
					Airline: "MH", Price: model.Money{Amount: 35000, Currency: "USD"},
				}, {
					From: "KUL", To: "MOW", SearchDate: "2026-11-14",
					Airline: "QR", Price: model.Money{Amount: 35000, Currency: "USD"},
				}},
			},
		}},
	}
	html := report.RenderBookingHTMLForTest(result)
	for _, want := range []string{
		"Лист для покупки",
		"кэш API · 12.07.2026 12:45 UTC",
		"MH",
		"aviasales.ru/search/MOW2609KTM13111",
		"aviasales.ru/search/MOW2609KUL1",
	} {
		if !strings.Contains(html, want) {
			t.Fatalf("missing %q in html", want)
		}
	}
}

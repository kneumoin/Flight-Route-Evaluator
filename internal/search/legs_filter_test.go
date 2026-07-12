package search

import (
	"testing"
	"time"

	"github.com/kneumoin/nepal/internal/config"
	"github.com/kneumoin/nepal/internal/model"
	"github.com/kneumoin/nepal/internal/scoring"
)

func TestFilterLeg2RejectsBeforeHubArrival(t *testing.T) {
	mow, _ := scoring.AirportLocation("MOW")
	dxb, _ := scoring.AirportLocation("DXB")
	dep := time.Date(2026, 9, 28, 8, 0, 0, 0, mow)
	arr := time.Date(2026, 9, 28, 14, 0, 0, 0, dxb)
	leg1 := []model.Offer{{
		TimingVerified: true,
		Segments: []model.Segment{{
			From: "MOW", To: "DXB", Departure: dep, Arrival: arr,
		}},
	}}
	leg2Early := model.Offer{
		SearchDate: "2026-09-26",
		Segments: []model.Segment{{
			From: "DXB", To: "KTM",
			Departure: time.Date(2026, 9, 26, 10, 0, 0, 0, dxb),
		}},
	}
	leg2OK := model.Offer{
		SearchDate: "2026-09-29",
		Segments: []model.Segment{{
			From: "DXB", To: "KTM",
			Departure: time.Date(2026, 9, 29, 10, 0, 0, 0, dxb),
		}},
	}
	out := filterLeg2AfterEarliest([]model.Offer{leg2Early, leg2OK}, leg1, 3)
	if len(out) != 1 || out[0].SearchDate != "2026-09-29" {
		t.Fatalf("filtered=%v", out)
	}
}

func TestLeg2SearchDates_AnchorsOnHubArrival(t *testing.T) {
	dxb, _ := scoring.AirportLocation("DXB")
	leg1 := []model.Offer{{
		Segments: []model.Segment{{
			From: "MOW", To: "DXB",
			Arrival: time.Date(2026, 9, 28, 14, 0, 0, 0, dxb),
		}},
	}}
	dates := leg2SearchDates("2026-09-28", leg1, config.BranchConfig{})
	if dates[0] != "2026-09-28" {
		t.Fatalf("dates=%v", dates)
	}
}

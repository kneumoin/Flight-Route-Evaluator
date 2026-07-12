package search

import (
	"testing"
	"time"

	"github.com/kneumoin/nepal/internal/config"
	"github.com/kneumoin/nepal/internal/model"
)

func TestLeg2SearchDates_CachedLeg1(t *testing.T) {
	leg1 := []model.Offer{{TimingVerified: false}}
	dates := leg2SearchDates("2026-09-28", leg1, config.BranchConfig{})
	want := []string{"2026-09-28", "2026-09-29", "2026-09-30"}
	if len(dates) != len(want) {
		t.Fatalf("dates=%v want=%v", dates, want)
	}
	for i := range want {
		if dates[i] != want[i] {
			t.Fatalf("dates=%v want=%v", dates, want)
		}
	}
}

func TestLeg2SearchDates_VerifiedArrival(t *testing.T) {
	arr := time.Date(2026, 9, 28, 21, 0, 0, 0, time.UTC)
	leg1 := []model.Offer{{
		TimingVerified: true,
		Segments: []model.Segment{{
			From: "MOW", To: "DXB", Arrival: arr,
		}},
	}}
	dates := leg2SearchDates("2026-09-28", leg1, config.BranchConfig{})
	has29 := false
	has30 := false
	for _, d := range dates {
		if d == "2026-09-29" {
			has29 = true
		}
		if d == "2026-09-30" {
			has30 = true
		}
	}
	if !has29 {
		t.Fatalf("expected 2026-09-29 in %v", dates)
	}
	if !has30 {
		t.Fatalf("expected +2 day for late arrival in %v", dates)
	}
}

func TestLegSearchDates_FirstLegUsesTripDate(t *testing.T) {
	dates := legSearchDates(0, "2026-09-28", nil, config.BranchConfig{}, nil)
	if len(dates) != 1 || dates[0] != "2026-09-28" {
		t.Fatalf("dates=%v", dates)
	}
}

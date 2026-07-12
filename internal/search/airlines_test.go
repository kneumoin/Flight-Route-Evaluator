package search_test

import (
	"testing"

	"github.com/kneumoin/nepal/internal/model"
	"github.com/kneumoin/nepal/internal/search"
)

func TestCollectLegAirlines(t *testing.T) {
	offers := []model.Offer{
		{SearchDate: "2026-09-28", AvailableAirlines: []string{"DP", "SU"}},
		{SearchDate: "2026-09-29", Segments: []model.Segment{{Airline: "FZ"}}},
	}
	got := search.CollectLegAirlinesForTest(offers, "2026-09-28")
	if len(got) != 2 {
		t.Fatalf("got %v", got)
	}
}

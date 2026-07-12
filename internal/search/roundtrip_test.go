package search_test

import (
	"testing"

	"github.com/kneumoin/nepal/internal/config"
	"github.com/kneumoin/nepal/internal/search"
)

func TestReverseBranch(t *testing.T) {
	b := config.BranchConfig{
		ID:   "via_doh",
		Name: "Via Doha",
		Legs: []config.LegConfig{
			{From: "MOW", To: "DOH"},
			{From: "DOH", To: "KTM"},
		},
	}
	rb := search.ReverseBranchForTest(b)
	if len(rb.Legs) != 2 {
		t.Fatalf("legs=%d", len(rb.Legs))
	}
	if rb.Legs[0].From != "KTM" || rb.Legs[0].To != "DOH" {
		t.Fatalf("leg0=%+v", rb.Legs[0])
	}
	if rb.Legs[1].From != "DOH" || rb.Legs[1].To != "MOW" {
		t.Fatalf("leg1=%+v", rb.Legs[1])
	}
}

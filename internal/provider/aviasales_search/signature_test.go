package aviasales_search

import "testing"

// Vector from Travelpayouts docs (How to create a signature md-5).
func TestBuildSignature_DocumentationExample(t *testing.T) {
	req := StartRequest{
		CurrencyCode: "USD",
		Marker:       "YourMarker",
		MarketCode:   "US",
		Locale:       "US",
		SearchParams: SearchParams{
			Directions: []Direction{
				{Origin: "LAX", Destination: "NYC", Date: "2026-09-09"},
				{Origin: "NYC", Destination: "LAX", Date: "2026-09-25"},
			},
			TripClass: "Y",
			Passengers: Passengers{Adults: 1, Children: 0, Infants: 0},
		},
	}
	parts := signatureParts(req)
	want := "USD:US:YourMarker:US:2026-09-09:NYC:LAX:2026-09-25:LAX:NYC:1:0:0:Y"
	got := joinColon(parts)
	if got != want {
		t.Fatalf("signature parts\ngot:  %s\nwant: %s", got, want)
	}
	sig := BuildSignature("YourToken", req)
	if sig == "" || len(sig) != 32 {
		t.Fatalf("unexpected sig %q", sig)
	}
	// Deterministic: same input → same hash
	if BuildSignature("YourToken", req) != sig {
		t.Fatal("signature not deterministic")
	}
}

func joinColon(parts []string) string {
	out := parts[0]
	for _, p := range parts[1:] {
		out += ":" + p
	}
	return out
}

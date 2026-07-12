package travelpayouts_data

import "testing"

func TestAirlinesInMonth(t *testing.T) {
	idx := monthPriceIndex{ByDate: map[string]dayPrice{
		"2026-09-08": {Airline: "FZ"},
		"2026-09-17": {Airline: "AI"},
		"2026-09-25": {Airline: "FZ"},
	}}
	got := idx.airlinesInMonth()
	if len(got) != 2 {
		t.Fatalf("airlines=%v", got)
	}
}

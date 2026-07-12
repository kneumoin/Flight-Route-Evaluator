package travelpayouts_data

import (
	"testing"
)

func TestAirlinesFromCheapRaw_Multiple(t *testing.T) {
	raw := []byte(`{"success":true,"data":{"DXB":{"0":{"price":720,"airline":"DP"},"1":{"price":800,"airline":"SU"}}}}`)
	got := airlinesFromCheapRaw(raw, "DXB")
	if len(got) != 2 {
		t.Fatalf("got %v", got)
	}
}

func TestMonthIndexAirlinesOnDate(t *testing.T) {
	idx := monthPriceIndex{ByDate: map[string]dayPrice{
		"2026-09-28": {Date: "2026-09-28", Airline: "QR", Price: 100},
	}}
	got := idx.airlinesOnDate("2026-09-28")
	if len(got) != 1 || got[0] != "QR" {
		t.Fatalf("got %v", got)
	}
}

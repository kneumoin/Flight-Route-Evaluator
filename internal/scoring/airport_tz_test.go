package scoring

import (
	"testing"
	"time"
)

func TestAirportLocation(t *testing.T) {
	loc, err := AirportLocation("KTM")
	if err != nil {
		t.Fatal(err)
	}
	if loc.String() != "Asia/Kathmandu" {
		t.Fatalf("got %s", loc)
	}
}

func TestConnectionDuration_CrossTimezone(t *testing.T) {
	mow, _ := AirportLocation("MOW")
	dxb, _ := AirportLocation("DXB")

	arrival := time.Date(2026, 9, 28, 18, 0, 0, 0, dxb)
	departure := time.Date(2026, 9, 29, 2, 0, 0, 0, dxb)
	d := ConnectionDuration(arrival, departure)
	if d != 8*time.Hour {
		t.Fatalf("expected 8h, got %v", d)
	}
	_ = mow
}

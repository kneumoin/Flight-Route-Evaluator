package mock

import (
	"context"
	"testing"

	"github.com/kneumoin/nepal/internal/model"
)

func TestMock_Search_KnownLeg(t *testing.T) {
	p := New()
	offers, err := p.Search(context.Background(), model.Query{From: "MOW", To: "DOH", Date: "2026-09-28"})
	if err != nil {
		t.Fatal(err)
	}
	if len(offers) != 1 || offers[0].Segments[0].Airline != "QR" {
		t.Fatalf("unexpected: %+v", offers)
	}
}

func TestMock_Search_UnknownLeg(t *testing.T) {
	p := New()
	offers, err := p.Search(context.Background(), model.Query{From: "MOW", To: "XXX"})
	if err != nil || len(offers) != 0 {
		t.Fatalf("expected empty, got %v err=%v", offers, err)
	}
}

func TestMock_Capabilities_SU(t *testing.T) {
	c := New().Capabilities()
	if !c.SupportedAirlines["SU"] {
		t.Fatal("mock should support SU")
	}
}

package travelpayouts_data

import (
	"testing"

	"github.com/kneumoin/nepal/internal/model"
)

func TestInspectLatest_DateFiltered(t *testing.T) {
	raw := []byte(`{"success":true,"data":[
		{"origin":"DXB","destination":"KTM","depart_date":"2026-09-17","value":169},
		{"origin":"DXB","destination":"KTM","depart_date":"2026-09-18","value":174}
	]}`)
	q := model.Query{From: "DXB", To: "KTM", Date: "2026-09-28"}
	stats := inspectLatestRaw(raw, q)
	if stats.RawOfferCount != 2 {
		t.Fatalf("raw=%d", stats.RawOfferCount)
	}
	if stats.AfterRouteFilter != 2 || stats.AfterDateFilter != 0 {
		t.Fatalf("route=%d date=%d", stats.AfterRouteFilter, stats.AfterDateFilter)
	}
	if stats.EmptyReason != EmptyDateFiltered {
		t.Fatalf("reason=%q", stats.EmptyReason)
	}
}

func TestInspectLatest_ExactDateMatch(t *testing.T) {
	raw := []byte(`{"success":true,"data":[
		{"origin":"DOH","destination":"KTM","depart_date":"2026-09-28","value":213}
	]}`)
	q := model.Query{From: "DOH", To: "KTM", Date: "2026-09-28"}
	stats := inspectLatestRaw(raw, q)
	if stats.ParsedOffers != 1 || stats.EmptyReason != EmptyNone {
		t.Fatalf("parsed=%d reason=%q", stats.ParsedOffers, stats.EmptyReason)
	}
}

func TestInspectCheap_APIEmpty(t *testing.T) {
	raw := []byte(`{"success":true,"data":{}}`)
	q := model.Query{From: "DXB", To: "KTM", Date: "2026-09-28"}
	stats := inspectCheapRaw(raw, q)
	if stats.EmptyReason != EmptyAPIEmpty {
		t.Fatalf("reason=%q", stats.EmptyReason)
	}
}

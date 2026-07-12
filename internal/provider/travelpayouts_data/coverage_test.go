package travelpayouts_data_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kneumoin/nepal/internal/model"
	"github.com/kneumoin/nepal/internal/provider/travelpayouts_data"
)

func TestParseCalendarResponse_Fixture(t *testing.T) {
	raw := mustReadFixture(t, "calendar_dxb_ktm.json")
	idx, err := travelpayouts_data.ParseCalendarResponseForTest(raw)
	if err != nil {
		t.Fatal(err)
	}
	if idx.CoverageDays() != 3 {
		t.Fatalf("days=%d", idx.CoverageDays())
	}
}

func TestCoverageFallback_DXBNearestDate(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.Contains(r.URL.Path, "cheap"):
			_, _ = w.Write([]byte(`{"success":true,"data":{}}`))
		case strings.Contains(r.URL.Path, "latest"):
			_, _ = w.Write([]byte(`{"success":true,"data":[]}`))
		case strings.Contains(r.URL.Path, "calendar"):
			_, _ = w.Write(mustReadFixture(t, "calendar_dxb_ktm.json"))
		case strings.Contains(r.URL.Path, "month-matrix"):
			_, _ = w.Write(mustReadFixture(t, "month_matrix_dxb_ktm.json"))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	p := newTestProvider(t, srv, testToken)
	q := model.Query{
		From: "DXB", To: "KTM", Date: "2026-09-28", TargetDate: "2026-09-28",
		AllowCoverageFallback: true, CoverageWindowDays: 3,
		NotBeforeDate: "2026-09-28",
	}
	offers, err := p.Search(context.Background(), q)
	if err != nil {
		t.Fatal(err)
	}
	if len(offers) != 1 {
		t.Fatalf("offers=%d", len(offers))
	}
	o := offers[0]
	if o.SearchDate != "2026-09-30" {
		t.Fatalf("selected=%s", o.SearchDate)
	}
	if !o.EstimatedDate {
		t.Fatalf("expected estimated date")
	}
}

func TestAnalyzeLegCoverage_Fixture(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "calendar") {
			_, _ = w.Write(mustReadFixture(t, "calendar_dxb_ktm.json"))
			return
		}
		if strings.Contains(r.URL.Path, "month-matrix") {
			_, _ = w.Write(mustReadFixture(t, "month_matrix_dxb_ktm.json"))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	p := newTestProvider(t, srv, testToken)
	row, err := p.AnalyzeLegCoverage(context.Background(), model.Query{
		From: "DXB", To: "KTM", TargetDate: "2026-09-28", CoverageWindowDays: 3,
	})
	if err != nil {
		t.Fatal(err)
	}
	if row.CoverageDays != 5 {
		t.Fatalf("coverage_days=%d", row.CoverageDays)
	}
	if row.CheapestDateNearTarget != "2026-09-25" {
		t.Fatalf("cheapest=%s", row.CheapestDateNearTarget)
	}
	if row.Airline != "FZ" {
		t.Fatalf("airline=%s", row.Airline)
	}
}

func mustReadFixture(t *testing.T, name string) []byte {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("..", "..", "..", "testdata", "travelpayouts", name))
	if err != nil {
		t.Fatal(err)
	}
	return b
}

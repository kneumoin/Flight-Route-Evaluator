package travelpayouts_data_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/kneumoin/nepal/internal/config"
	"github.com/kneumoin/nepal/internal/model"
	"github.com/kneumoin/nepal/internal/provider/travelpayouts_data"
	"github.com/kneumoin/nepal/internal/secrets"
)

const testToken = "test-token"

func TestSearch_MissingToken(t *testing.T) {
	p := travelpayouts_data.New(config.CacheConfig{Enabled: false}, "", "USD", false)
	_, err := p.Search(context.Background(), model.Query{From: "MOW", To: "KTM", Date: "2026-09-28"})
	if err == nil || !strings.Contains(err.Error(), "TRAVELPAYOUTS_TOKEN") {
		t.Fatalf("expected missing token error, got %v", err)
	}
}

func TestSearch_ParsesCheapResponse(t *testing.T) {
	var calls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if got := r.Header.Get("X-Access-Token"); got != testToken {
			t.Fatalf("unexpected token header: %q", got)
		}
		if strings.Contains(r.URL.RawQuery, testToken) {
			t.Fatal("token must not appear in URL")
		}
		if strings.Contains(r.URL.Path, "cheap") {
			_, _ = w.Write([]byte(`{"success":true,"data":{"KTM":{"0":{"price":850,"airline":"QR","flight_number":100,"departure_at":"2026-09-28T08:00:00Z"}}}}`))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	p := newTestProvider(t, srv, testToken)
	offers, err := p.Search(context.Background(), model.Query{From: "MOW", To: "KTM", Date: "2026-09-28"})
	if err != nil {
		t.Fatal(err)
	}
	if len(offers) != 1 {
		t.Fatalf("expected 1 offer, got %d", len(offers))
	}
	o := offers[0]
	if o.Provider != "travelpayouts_data" || o.DataQuality != model.DataQualityCached {
		t.Fatalf("unexpected offer meta: %+v", o)
	}
	if o.Price.Amount != 85000 || o.CheckedBaggageKg != nil {
		t.Fatalf("unexpected price/baggage: %+v", o)
	}
	if o.Segments[0].Airline != "QR" {
		t.Fatalf("expected airline QR, got %q", o.Segments[0].Airline)
	}
	if calls != 1 {
		t.Fatalf("expected 1 HTTP call, got %d", calls)
	}
}

func TestSearch_CacheHit(t *testing.T) {
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		_, _ = w.Write([]byte(`{"success":true,"data":{"KTM":{"0":{"price":900,"airline":"SU","flight_number":200}}}}`))
	}))
	defer srv.Close()

	p := newTestProvider(t, srv, testToken)
	q := model.Query{From: "MOW", To: "KTM", Date: "2026-09-28"}
	if _, err := p.Search(context.Background(), q); err != nil {
		t.Fatal(err)
	}
	if _, err := p.Search(context.Background(), q); err != nil {
		t.Fatal(err)
	}
	if calls != 1 {
		t.Fatalf("expected cache hit on second call, HTTP calls=%d", calls)
	}
}

func TestSearch_Non2xxIsError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"error":"forbidden"}`))
	}))
	defer srv.Close()

	p := newTestProvider(t, srv, testToken)
	_, err := p.Search(context.Background(), model.Query{From: "MOW", To: "KTM", Date: "2026-09-28"})
	if err == nil || !strings.Contains(err.Error(), "403") {
		t.Fatalf("expected status error, got %v", err)
	}
}

func TestSecretsRedact(t *testing.T) {
	token := "abc123def456"
	s := secrets.Redact("GET https://example.com?token="+token, token)
	if strings.Contains(s, token) {
		t.Fatalf("token not redacted: %s", s)
	}
	if !strings.Contains(s, secrets.Redacted) {
		t.Fatal("expected REDACTED marker")
	}
}

func newTestProvider(t *testing.T, srv *httptest.Server, token string) *travelpayouts_data.Provider {
	t.Helper()
	dir := t.TempDir()
	p := travelpayouts_data.New(config.CacheConfig{
		Enabled: true, TTL: "1h", Directory: dir,
	}, token, "USD", false)
	p.SetAPIRoot(srv.URL)
	p.SetHTTPClient(srv.Client())
	return p
}

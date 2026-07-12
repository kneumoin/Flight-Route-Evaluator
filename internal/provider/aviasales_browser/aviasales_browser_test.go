package aviasales_browser_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/kneumoin/nepal/internal/model"
	"github.com/kneumoin/nepal/internal/provider/aviasales_browser"
)

func TestSearchURL(t *testing.T) {
	url, err := aviasales_browser.SearchURL("MOW", "KTM", "2026-09-28", 1)
	if err != nil {
		t.Fatal(err)
	}
	want := "https://www.aviasales.ru/search/MOW2809KTM1"
	if url != want {
		t.Fatalf("got %q want %q", url, want)
	}
}

func TestParseOffersFixture(t *testing.T) {
	html := readFixture(t, "offers.html")
	offers, err := aviasales_browser.ParseHTMLForTest(html, "MOW", "KTM", "2026-09-28")
	if err != nil {
		t.Fatal(err)
	}
	if len(offers) != 2 {
		t.Fatalf("expected 2 offers, got %d", len(offers))
	}
	if offers[0].PriceAmount != 85000 {
		t.Fatalf("expected cheapest parsed first pass, got %d", offers[0].PriceAmount)
	}
	if offers[0].BookingURL != "https://agency.example/book/qr850" {
		t.Fatalf("booking url=%q", offers[0].BookingURL)
	}
}

func TestParseRealAviasalesFixture(t *testing.T) {
	html := readFixture(t, "real_offers.html")
	offers, err := aviasales_browser.ParseHTMLForTest(html, "MOW", "KTM", "2026-09-26")
	if err != nil {
		t.Fatalf("real fixture must parse (reCAPTCHA lib present but not a wall): %v", err)
	}
	if len(offers) != 2 {
		t.Fatalf("expected 2 offers, got %d", len(offers))
	}
	// Narrow no-break space thousands separator must be handled.
	if offers[0].PriceAmount != 7726000 {
		t.Fatalf("price0=%d want 7726000", offers[0].PriceAmount)
	}
	if offers[0].Currency != "RUB" {
		t.Fatalf("currency0=%q", offers[0].Currency)
	}
	if offers[0].Airline != "China Southern Airlines" {
		t.Fatalf("airline0=%q", offers[0].Airline)
	}
	if offers[0].DepartureClock != "16:05" || offers[0].ArrivalClock != "11:30" {
		t.Fatalf("times0=%s-%s", offers[0].DepartureClock, offers[0].ArrivalClock)
	}
	if offers[0].Stops != 1 {
		t.Fatalf("stops0=%d want 1", offers[0].Stops)
	}
	if len(offers[1].Airlines) != 2 {
		t.Fatalf("offer1 airlines=%v want 2 carriers", offers[1].Airlines)
	}
}

func TestParseCaptchaFixture(t *testing.T) {
	html := readFixture(t, "captcha.html")
	_, err := aviasales_browser.ParseHTMLForTest(html, "MOW", "KTM", "2026-09-28")
	if err == nil || !strings.Contains(err.Error(), "captcha") {
		t.Fatalf("expected captcha error, got %v", err)
	}
}

func TestSearch_CacheHitSkipsFetcher(t *testing.T) {
	url, _ := aviasales_browser.SearchURL("MOW", "KTM", "2026-09-28", 1)
	dir := t.TempDir()

	fetcher := &aviasales_browser.StaticFetcher{Pages: map[string]string{
		url: readFixture(t, "offers.html"),
	}}

	opts := aviasales_browser.DefaultOptions()
	opts.CacheEnabled = true
	opts.CacheDir = dir
	opts.CacheTTL = "1h"
	opts.RateLimit = 0

	p := aviasales_browser.New(opts, fetcher)
	if _, err := p.Search(context.Background(), model.Query{From: "MOW", To: "KTM", Date: "2026-09-28", Passengers: 1}); err != nil {
		t.Fatal(err)
	}

	// Replace fetcher that would fail if called
	p.SetFetcher(&aviasales_browser.StaticFetcher{Pages: map[string]string{}})
	offers, err := p.Search(context.Background(), model.Query{From: "MOW", To: "KTM", Date: "2026-09-28", Passengers: 1})
	if err != nil {
		t.Fatal(err)
	}
	if len(offers) != 1 {
		t.Fatalf("expected 1 offer from cache, got %d", len(offers))
	}
	if offers[0].DataQuality != model.DataQualityBrowserCollected {
		t.Fatalf("unexpected data quality %q", offers[0].DataQuality)
	}
}

func TestSearch_CacheOnlyMiss(t *testing.T) {
	opts := aviasales_browser.DefaultOptions()
	opts.CacheEnabled = true
	opts.CacheDir = t.TempDir()
	opts.CacheOnly = true
	p := aviasales_browser.New(opts, &aviasales_browser.StaticFetcher{Pages: map[string]string{}})
	_, err := p.Search(context.Background(), model.Query{From: "MOW", To: "KTM", Date: "2026-09-28", Passengers: 1})
	if err == nil || !strings.Contains(err.Error(), "cache miss") {
		t.Fatalf("expected cache miss error, got %v", err)
	}
}

func TestProviderCapabilities(t *testing.T) {
	p := aviasales_browser.New(aviasales_browser.DefaultOptions(), &aviasales_browser.StaticFetcher{})
	c := p.Capabilities()
	if c.SupportsRealTimePricing {
		t.Fatal("browser provider must not claim realtime pricing")
	}
	if c.AirlineCoverageMode != model.CoverageUnknown {
		t.Fatalf("expected unknown coverage, got %q", c.AirlineCoverageMode)
	}
}

func readFixture(t *testing.T, name string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestRateLimiter_Serial(t *testing.T) {
	opts := aviasales_browser.DefaultOptions()
	opts.RateLimit = 50 * time.Millisecond
	opts.CacheEnabled = false
	url1, _ := aviasales_browser.SearchURL("MOW", "DOH", "2026-09-28", 1)
	url2, _ := aviasales_browser.SearchURL("MOW", "IST", "2026-09-28", 1)
	html := readFixture(t, "offers.html")
	fetcher := &aviasales_browser.StaticFetcher{Pages: map[string]string{url1: html, url2: html}}
	p := aviasales_browser.New(opts, fetcher)
	start := time.Now()
	if _, err := p.Search(context.Background(), model.Query{From: "MOW", To: "DOH", Date: "2026-09-28", Passengers: 1}); err != nil {
		t.Fatal(err)
	}
	if _, err := p.Search(context.Background(), model.Query{From: "MOW", To: "IST", Date: "2026-09-28", Passengers: 1}); err != nil {
		t.Fatal(err)
	}
	if elapsed := time.Since(start); elapsed < 45*time.Millisecond {
		t.Fatalf("expected rate limit delay between fetches, elapsed %v", elapsed)
	}
}

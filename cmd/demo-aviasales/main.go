// UI-only demo for Aviasales integration (no Search API).
//
// Usage:
//
//	go run ./cmd/demo-aviasales help
//	go run ./cmd/demo-aviasales urls -from MOW -to KTM -date 2026-09-26 -return 2026-11-13
//	go run ./cmd/demo-aviasales parse-fixture
//	go run ./cmd/demo-aviasales browser -from MOW -to KTM -date 2026-09-26
//	go run ./cmd/demo-aviasales inspect -from MOW -to KTM -date 2026-09-26 -save .cache/aviasales_debug.html
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/kneumoin/nepal/internal/links"
	"github.com/kneumoin/nepal/internal/model"
	"github.com/kneumoin/nepal/internal/provider/aviasales_browser"
)

func main() {
	if len(os.Args) < 2 {
		printHelp()
		os.Exit(0)
	}
	switch os.Args[1] {
	case "urls":
		runURLs(os.Args[2:])
	case "parse-fixture":
		runParseFixture()
	case "browser":
		runBrowser(os.Args[2:])
	case "inspect":
		runInspect(os.Args[2:])
	case "help", "-h", "--help":
		printHelp()
	default:
		fmt.Fprintf(os.Stderr, "unknown subcommand %q\n\n", os.Args[1])
		printHelp()
		os.Exit(2)
	}
}

func printHelp() {
	fmt.Print(`demo-aviasales — Aviasales через UI (без API)

Модель драйвера:
  1. Собрать search URL (как пользователь открывает страницу)
  2. Chrome (headful) загружает страницу, ждёт карточки билетов
  3. parseHTML вытаскивает цену / авиакомпанию / ссылку «Купить»
  4. Кэш HTML + результат в .cache/aviasales_browser/

Команды:
  urls          URL для ручной проверки (без Chrome)
  parse-fixture разбор тестового HTML (offline)
  browser       полный цикл: URL → Chrome → offers
  inspect       открыть страницу, сохранить HTML, подсказки по селекторам

Примеры:
  go run ./cmd/demo-aviasales inspect -from MOW -to KTM -date 2026-09-26 -headful=true
  go run ./cmd/demo-aviasales browser -from MOW -to KUL -date 2026-09-26

Round-trip: отдельный search URL (MOW2609KTM13111) или два OW-поиска на каждый leg.

Пакет драйвера: internal/provider/aviasales_browser/
`)
}

func runURLs(args []string) {
	fs := flag.NewFlagSet("urls", flag.ExitOnError)
	from := fs.String("from", "MOW", "origin IATA")
	to := fs.String("to", "KTM", "destination IATA")
	date := fs.String("date", "2026-09-26", "departure YYYY-MM-DD")
	ret := fs.String("return", "", "return YYYY-MM-DD (optional)")
	pax := fs.Int("pax", 1, "passengers")
	_ = fs.Parse(args)

	ow, err := links.OneWaySearchURL(*from, *to, *date, *pax)
	if err != nil {
		fatal(err)
	}
	fmt.Println("One-way:", ow)
	if *ret != "" {
		rt, err := links.RoundTripShortURL(*from, *to, *date, *ret, *pax)
		if err != nil {
			fatal(err)
		}
		fmt.Println("Round-trip:", rt)
	}
	fmt.Println("\nОткройте URL в браузере — так же работает наш Chrome-драйвер.")
}

func runParseFixture() {
	fixture := filepath.Join("internal", "provider", "aviasales_browser", "testdata", "offers.html")
	raw, err := os.ReadFile(fixture)
	if err != nil {
		fatal(err)
	}
	offers, err := aviasales_browser.ParseHTMLForTest(string(raw), "MOW", "KTM", "2026-09-28")
	if err != nil {
		fatal(err)
	}
	fmt.Printf("Parsed %d offers\n", len(offers))
	for i, o := range offers {
		fmt.Printf("[%d] %s $%.0f booking=%q search=%q\n",
			i+1, o.Airline, float64(o.PriceAmount)/100, o.BookingURL, o.SearchPageURL)
	}
}

func runBrowser(args []string) {
	fs := flag.NewFlagSet("browser", flag.ExitOnError)
	from := fs.String("from", "MOW", "origin")
	to := fs.String("to", "KTM", "destination")
	date := fs.String("date", "2026-09-26", "departure")
	headful := fs.Bool("headful", true, "visible Chrome")
	timeout := fs.Duration("timeout", 120*time.Second, "timeout")
	_ = fs.Parse(args)

	url, err := aviasales_browser.SearchURL(*from, *to, *date, 1)
	if err != nil {
		fatal(err)
	}
	fmt.Println("Search URL:", url)

	opts := aviasales_browser.DefaultOptions()
	opts.Headful = *headful
	opts.Timeout = *timeout
	opts.Verbose = true
	opts.CacheEnabled = true
	p := aviasales_browser.New(opts, nil)

	offers, err := p.Search(context.Background(), model.Query{
		From: *from, To: *to, Date: *date, Passengers: 1,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "\nFailed: %v\n", err)
		fmt.Fprintln(os.Stderr, "Tip: run `inspect` first to save HTML and tune selectors.")
		os.Exit(1)
	}
	fmt.Printf("\n%d offer(s)\n", len(offers))
	for _, o := range offers {
		airline := ""
		if len(o.Segments) > 0 {
			airline = o.Segments[0].Airline
		}
		fmt.Printf("  $%.0f %s  airline=%s\n", float64(o.Price.Amount)/100, o.Price.Currency, airline)
	}
}

func runInspect(args []string) {
	fs := flag.NewFlagSet("inspect", flag.ExitOnError)
	from := fs.String("from", "MOW", "origin")
	to := fs.String("to", "KTM", "destination")
	date := fs.String("date", "2026-09-26", "departure")
	headful := fs.Bool("headful", true, "visible Chrome")
	timeout := fs.Duration("timeout", 120*time.Second, "timeout")
	save := fs.String("save", ".cache/aviasales_inspect.html", "where to write HTML")
	_ = fs.Parse(args)

	url, err := aviasales_browser.SearchURL(*from, *to, *date, 1)
	if err != nil {
		fatal(err)
	}
	fmt.Println("Opening:", url)

	fetcher := &aviasales_browser.ChromeFetcher{Headful: *headful, Timeout: *timeout, Verbose: true}
	html, err := fetcher.Fetch(context.Background(), url)
	if err != nil {
		fatal(err)
	}

	if err := os.MkdirAll(filepath.Dir(*save), 0o755); err != nil {
		fatal(err)
	}
	if err := os.WriteFile(*save, []byte(html), 0o644); err != nil {
		fatal(err)
	}
	fmt.Println("Saved HTML:", *save, "(", len(html), "bytes)")

	hint := aviasales_browser.InspectHTML(html)
	fmt.Println("\n--- DOM hints ---")
	fmt.Println("title:", hint.Title)
	fmt.Println("captcha:", hint.CaptchaDetected)
	fmt.Println("offer-like nodes (current selectors):", hint.OfferLikeCount)
	fmt.Println("http links found:", hint.LinkCount)
	if len(hint.SampleClasses) > 0 {
		fmt.Println("\nClasses to investigate:")
		for _, c := range hint.SampleClasses {
			fmt.Println(" ", c)
		}
	}
	if len(hint.SampleLinks) > 0 {
		fmt.Println("\nSample links:")
		for _, l := range hint.SampleLinks {
			fmt.Println(" ", l)
		}
	}

	offers, parseErr := aviasales_browser.ParseHTMLForTest(html, *from, *to, *date)
	if parseErr != nil {
		fmt.Println("\nParser:", parseErr)
		fmt.Println("→ Update selectorOfferCard in selectors.go using saved HTML.")
	} else {
		fmt.Println("\nParser OK:", len(offers), "offers")
		for i, o := range offers {
			fmt.Printf("  [%d] %s $%.0f booking=%q\n", i+1, o.Airline, float64(o.PriceAmount)/100, o.BookingURL)
		}
	}
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "error:", err)
	os.Exit(1)
}

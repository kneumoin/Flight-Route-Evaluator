package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/kneumoin/nepal/internal/config"
	"github.com/kneumoin/nepal/internal/model"
	"github.com/kneumoin/nepal/internal/provider/travelpayouts_data"
)

func main() {
	from := flag.String("from", "", "origin IATA (required)")
	to := flag.String("to", "", "destination IATA (required)")
	date := flag.String("date", "2026-09-28", "departure date YYYY-MM-DD")
	noCache := flag.Bool("no-cache", true, "bypass response cache for fresh API data")
	flag.Parse()

	if *from == "" || *to == "" {
		fmt.Fprintln(os.Stderr, "usage: debug-leg -from DXB -to KTM [-date 2026-09-28] [-no-cache=true]")
		os.Exit(2)
	}

	token := os.Getenv("TRAVELPAYOUTS_TOKEN")
	if token == "" {
		fmt.Fprintln(os.Stderr, "TRAVELPAYOUTS_TOKEN not set")
		os.Exit(1)
	}

	cacheCfg := config.CacheConfig{Enabled: !*noCache, TTL: "1h", Directory: ".cache"}
	p := travelpayouts_data.New(cacheCfg, token, "USD", true)

	q := model.Query{From: strings.ToUpper(*from), To: strings.ToUpper(*to), Date: *date}
	offers, err := p.Search(context.Background(), q)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("\noffers returned: %d\n", len(offers))
	if len(offers) > 0 {
		o := offers[0]
		fmt.Printf("  price: %.2f %s  provider: %s  quality: %s\n",
			float64(o.Price.Amount)/100, o.Price.Currency, o.Provider, o.DataQuality)
	}
}

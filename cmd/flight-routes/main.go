package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/kneumoin/nepal/internal/config"
	"github.com/kneumoin/nepal/internal/provider"
	"github.com/kneumoin/nepal/internal/provider/aviasales"
	"github.com/kneumoin/nepal/internal/provider/aviasales_browser"
	"github.com/kneumoin/nepal/internal/provider/kiwi"
	"github.com/kneumoin/nepal/internal/provider/mock"
	"github.com/kneumoin/nepal/internal/provider/travelpayouts_data"
	"github.com/kneumoin/nepal/internal/report"
	"github.com/kneumoin/nepal/internal/search"
	"github.com/kneumoin/nepal/internal/secrets"
)

func main() {
	configPath := flag.String("config", "", "path to YAML config")
	outDir := flag.String("out", "./out", "output directory")
	providerMode := flag.String("provider", "", "force provider: mock, travelpayouts_data, aviasales_browser")
	travelpayoutsToken := flag.String("travelpayouts-token", "", "Travelpayouts API token (overrides TRAVELPAYOUTS_TOKEN; never logged)")
	browserHeadful := flag.Bool("browser-headful", true, "show visible browser for aviasales_browser provider")
	browserRateLimit := flag.Duration("browser-rate-limit", time.Minute, "min interval between browser page loads")
	browserCacheOnly := flag.Bool("browser-cache-only", false, "only use cached aviasales_browser results")
	browserTimeout := flag.Duration("browser-timeout", 120*time.Second, "browser page load timeout")
	verbose := flag.Bool("verbose", false, "verbose API debug logging")
	quiet := flag.Bool("quiet", false, "suppress progress output on stderr")
	finalists := flag.Int("finalists", report.DefaultFinalists, "how many top routes to put in finalists.html")
	flag.Parse()

	if *configPath == "" {
		fmt.Fprintln(os.Stderr, "--config is required")
		os.Exit(2)
	}

	cfg, err := config.Load(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "config error: %v\n", err)
		os.Exit(1)
	}

	token := secrets.TravelpayoutsToken(*travelpayoutsToken)
	mockOnly := *providerMode == "mock"
	forcedData := *providerMode == "travelpayouts_data"
	forcedBrowser := *providerMode == "aviasales_browser"

	reg := provider.NewRegistry()
	reg.Register(mock.New())

	if mockOnly {
		cfg.Providers = []config.ProviderConfig{{ID: "mock", Enabled: true}}
	} else if forcedBrowser {
		bopts := aviasales_browser.DefaultOptions()
		bopts.Headful = *browserHeadful
		bopts.RateLimit = *browserRateLimit
		bopts.CacheOnly = *browserCacheOnly
		bopts.Timeout = *browserTimeout
		bopts.Verbose = *verbose
		bopts.CacheEnabled = cfg.Cache.Enabled
		bopts.CacheDir = cfg.Cache.Directory
		bopts.CacheTTL = cfg.Cache.TTL
		reg.Register(aviasales_browser.New(bopts, nil))
		cfg.Providers = []config.ProviderConfig{{ID: "aviasales_browser", Enabled: true}}
		if *verbose {
			log.Printf("aviasales_browser: experimental local-only mode (headful=%v rate=%s cache-only=%v)",
				*browserHeadful, *browserRateLimit, *browserCacheOnly)
		}
	} else {
		if !forcedData {
			reg.Register(aviasales.New(cfg.Cache))
			reg.Register(kiwi.New())
		}
		reg.Register(travelpayouts_data.New(cfg.Cache, token, cfg.Scoring.Currency, *verbose))
		if forcedData {
			cfg.Providers = []config.ProviderConfig{{ID: "travelpayouts_data", Enabled: true}}
			if token == "" && *verbose {
				log.Printf("warn: TRAVELPAYOUTS_TOKEN missing, travelpayouts_data provider unavailable")
			}
		} else {
			cfg.Providers = filterProviders(cfg.Providers, reg, token, *verbose)
		}
	}

	eval := &search.Evaluator{
		Config: cfg, Registry: reg, Verbose: *verbose, Progress: !*quiet, MockOnly: mockOnly,
	}
	runStart := time.Now()
	result, err := eval.Evaluate(context.Background())
	if err != nil {
		log.Fatalf("evaluate: %v", err)
	}

	if err := report.WriteHTML(filepath.Join(*outDir, "report.html"), result); err != nil {
		log.Fatalf("html: %v", err)
	}
	if err := report.WriteCSV(filepath.Join(*outDir, "report.csv"), result); err != nil {
		log.Fatalf("csv: %v", err)
	}
	if err := report.WriteJSON(filepath.Join(*outDir, "results.json"), result); err != nil {
		log.Fatalf("json: %v", err)
	}
	if err := report.WriteBookingHTML(filepath.Join(*outDir, "booking.html"), result); err != nil {
		log.Fatalf("booking html: %v", err)
	}
	if err := report.WriteBookingCSV(filepath.Join(*outDir, "booking.csv"), result); err != nil {
		log.Fatalf("booking csv: %v", err)
	}
	if err := report.WriteFinalistsHTML(filepath.Join(*outDir, "finalists.html"), result, *finalists); err != nil {
		log.Fatalf("finalists html: %v", err)
	}

	eval.PrintProgressSummary(result.Branches, *outDir, time.Since(runStart))

	if *verbose {
		log.Printf("wrote reports to %s", *outDir)
	}
}

func filterProviders(providers []config.ProviderConfig, reg *provider.Registry, travelpayoutsToken string, verbose bool) []config.ProviderConfig {
	out := make([]config.ProviderConfig, 0, len(providers))
	for _, p := range providers {
		if !p.Enabled {
			continue
		}
		if p.ID == "mock" || p.ID == "aviasales_browser" {
			continue
		}
		if _, ok := reg.Get(p.ID); !ok {
			if verbose {
				log.Printf("warn: provider %s not registered, skipping", p.ID)
			}
			continue
		}
		if p.ID == "kiwi" && os.Getenv("KIWI_API_KEY") == "" {
			if verbose {
				log.Printf("warn: KIWI_API_KEY missing, skipping kiwi")
			}
			continue
		}
		if p.ID == "aviasales" && os.Getenv("AVIASALES_TOKEN") == "" && travelpayoutsToken == "" {
			if verbose {
				log.Printf("warn: AVIASALES_TOKEN missing, skipping aviasales")
			}
			continue
		}
		if p.ID == "travelpayouts_data" && travelpayoutsToken == "" {
			if verbose {
				log.Printf("warn: TRAVELPAYOUTS_TOKEN missing, skipping travelpayouts_data")
			}
			continue
		}
		out = append(out, p)
	}
	return out
}

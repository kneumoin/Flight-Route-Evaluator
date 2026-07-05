package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/kneumoin/nepal/internal/config"
	"github.com/kneumoin/nepal/internal/provider"
	"github.com/kneumoin/nepal/internal/provider/aviasales"
	"github.com/kneumoin/nepal/internal/provider/kiwi"
	"github.com/kneumoin/nepal/internal/provider/mock"
	"github.com/kneumoin/nepal/internal/report"
	"github.com/kneumoin/nepal/internal/search"
)

func main() {
	configPath := flag.String("config", "", "path to YAML config")
	outDir := flag.String("out", "./out", "output directory")
	providerMode := flag.String("provider", "", "force provider mode: mock")
	verbose := flag.Bool("verbose", false, "verbose logging")
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

	mockOnly := *providerMode == "mock"

	reg := provider.NewRegistry()
	reg.Register(mock.New())
	if !mockOnly {
		reg.Register(aviasales.New(cfg.Cache))
		reg.Register(kiwi.New())
		cfg.Providers = filterProviders(cfg.Providers, reg, *verbose)
	} else {
		cfg.Providers = []config.ProviderConfig{{ID: "mock", Enabled: true}}
	}

	eval := &search.Evaluator{
		Config: cfg, Registry: reg, Verbose: *verbose, MockOnly: mockOnly,
	}
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

	if *verbose {
		log.Printf("wrote reports to %s", *outDir)
	}
}

func filterProviders(providers []config.ProviderConfig, reg *provider.Registry, verbose bool) []config.ProviderConfig {
	out := make([]config.ProviderConfig, 0, len(providers))
	for _, p := range providers {
		if !p.Enabled {
			continue
		}
		if p.ID == "mock" {
			continue
		}
		if _, ok := reg.Get(p.ID); !ok {
			if verbose {
				log.Printf("warn: provider %s not registered, skipping", p.ID)
			}
			continue
		}
		// Skip kiwi when no key (stub)
		if p.ID == "kiwi" && os.Getenv("KIWI_API_KEY") == "" {
			if verbose {
				log.Printf("warn: KIWI_API_KEY missing, skipping kiwi")
			}
			continue
		}
		if p.ID == "aviasales" && os.Getenv("AVIASALES_TOKEN") == "" {
			if verbose {
				log.Printf("warn: AVIASALES_TOKEN missing, skipping aviasales")
			}
			continue
		}
		out = append(out, p)
	}
	return out
}

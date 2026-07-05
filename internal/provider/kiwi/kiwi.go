package kiwi

import (
	"context"
	"fmt"
	"os"

	"github.com/kneumoin/nepal/internal/model"
	"github.com/kneumoin/nepal/internal/provider"
)

// Provider is a stub until Kiwi API access is configured.
type Provider struct {
	apiKey string
}

func New() *Provider {
	return &Provider{apiKey: os.Getenv("KIWI_API_KEY")}
}

func (p *Provider) Name() string { return "kiwi" }

func (p *Provider) Capabilities() provider.ProviderCapabilities {
	return provider.ProviderCapabilities{
		SupportedAirlines: map[string]bool{
			"FZ": true, "6E": true,
		},
		AirlineCoverageMode:     model.CoveragePartial,
		SupportsSelfTransfer:    true,
		SupportsBaggageInfo:     true,
		SupportsRealTimePricing: true,
	}
}

func (p *Provider) Search(ctx context.Context, q model.Query) ([]model.Offer, error) {
	_ = ctx
	_ = q
	if p.apiKey == "" {
		return nil, fmt.Errorf("KIWI_API_KEY not set; kiwi provider disabled")
	}
	return nil, fmt.Errorf("kiwi API integration not implemented")
}

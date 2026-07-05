package provider

import (
	"context"

	"github.com/kneumoin/nepal/internal/model"
)

type Provider interface {
	Name() string
	Capabilities() ProviderCapabilities
	Search(ctx context.Context, query model.Query) ([]model.Offer, error)
}

type ProviderCapabilities struct {
	SupportedAirlines       map[string]bool
	AirlineCoverageMode     model.AirlineCoverageMode
	SupportsSelfTransfer    bool
	SupportsBaggageInfo     bool
	SupportsRealTimePricing bool
}

type Registry struct {
	providers map[string]Provider
	order     []string
}

func NewRegistry() *Registry {
	return &Registry{providers: make(map[string]Provider)}
}

func (r *Registry) Register(p Provider) {
	id := p.Name()
	if _, exists := r.providers[id]; !exists {
		r.order = append(r.order, id)
	}
	r.providers[id] = p
}

func (r *Registry) Get(id string) (Provider, bool) {
	p, ok := r.providers[id]
	return p, ok
}

func (r *Registry) All() []Provider {
	out := make([]Provider, 0, len(r.order))
	for _, id := range r.order {
		out = append(out, r.providers[id])
	}
	return out
}

func (c ProviderCapabilities) SupportsAirline(code string) bool {
	if c.AirlineCoverageMode == model.CoverageUnknown || c.AirlineCoverageMode == model.CoveragePartial {
		return true // advisory only
	}
	return c.SupportedAirlines[code]
}

func (c ProviderCapabilities) DeclaresSU() bool {
	if c.AirlineCoverageMode == model.CoverageUnknown || c.AirlineCoverageMode == model.CoveragePartial {
		return c.SupportedAirlines["SU"]
	}
	return c.SupportedAirlines["SU"]
}

package provider

import (
	"context"
	"testing"

	"github.com/kneumoin/nepal/internal/config"
	"github.com/kneumoin/nepal/internal/model"
)

type stubProvider struct {
	id   string
	caps ProviderCapabilities
}

func (s stubProvider) Name() string { return s.id }
func (s stubProvider) Capabilities() ProviderCapabilities { return s.caps }
func (s stubProvider) Search(ctx context.Context, q model.Query) ([]model.Offer, error) {
	return nil, nil
}

func TestSelectProvider_HintWins(t *testing.T) {
	reg := NewRegistry()
	reg.Register(stubProvider{id: "aviasales", caps: ProviderCapabilities{SupportedAirlines: map[string]bool{"SU": true}, AirlineCoverageMode: model.CoverageKnown}})
	reg.Register(stubProvider{id: "kiwi", caps: ProviderCapabilities{SupportedAirlines: map[string]bool{"FZ": true}, AirlineCoverageMode: model.CoverageKnown}})
	leg := config.LegConfig{From: "MOW", To: "DXB", PreferredAirlines: []string{"SU"}, ProviderHint: "kiwi"}
	enabled := []config.ProviderConfig{{ID: "aviasales", Enabled: true}, {ID: "kiwi", Enabled: true}}
	p, ok := SelectProvider(leg, enabled, reg)
	if !ok || p.Name() != "kiwi" {
		t.Fatalf("expected kiwi hint, got %v ok=%v", p, ok)
	}
}

func TestSelectProvider_PartialNotRejected(t *testing.T) {
	reg := NewRegistry()
	reg.Register(stubProvider{id: "unknown", caps: ProviderCapabilities{AirlineCoverageMode: model.CoverageUnknown}})
	leg := config.LegConfig{From: "MOW", To: "KTM", PreferredAirlines: []string{"XX"}}
	enabled := []config.ProviderConfig{{ID: "unknown", Enabled: true}}
	p, ok := SelectProvider(leg, enabled, reg)
	if !ok || p == nil {
		t.Fatal("unknown coverage provider should remain selectable")
	}
}

func TestSUProviderUnavailable(t *testing.T) {
	reg := NewRegistry()
	reg.Register(stubProvider{id: "kiwi", caps: ProviderCapabilities{SupportedAirlines: map[string]bool{"FZ": true}, AirlineCoverageMode: model.CoverageKnown}})
	enabled := []config.ProviderConfig{{ID: "kiwi", Enabled: true}}
	if SUProviderAvailable([]string{"SU"}, enabled, reg) {
		t.Fatal("expected no SU provider")
	}
}

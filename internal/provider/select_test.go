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

func (s stubProvider) Name() string                            { return s.id }
func (s stubProvider) Capabilities() ProviderCapabilities      { return s.caps }
func (s stubProvider) Search(ctx context.Context, q model.Query) ([]model.Offer, error) {
	return nil, nil
}

func TestSelectProvider_HintWinsWhenEnabled(t *testing.T) {
	reg := NewRegistry()
	reg.Register(stubProvider{id: "aviasales", caps: ProviderCapabilities{SupportedAirlines: map[string]bool{"SU": true}, AirlineCoverageMode: model.CoverageKnown}})
	reg.Register(stubProvider{id: "kiwi", caps: ProviderCapabilities{SupportedAirlines: map[string]bool{"FZ": true}, AirlineCoverageMode: model.CoverageKnown}})
	leg := config.LegConfig{From: "MOW", To: "DXB", PreferredAirlines: []string{"FZ"}, ProviderHint: "kiwi"}
	enabled := []config.ProviderConfig{{ID: "aviasales", Enabled: true}, {ID: "kiwi", Enabled: true}}
	p, ok := SelectProvider(leg, enabled, reg, config.AirlinePreferenceAdvisory)
	if !ok || p.Name() != "kiwi" {
		t.Fatalf("expected kiwi hint, got %v ok=%v", p, ok)
	}
}

func TestSelectProvider_DisabledHintFallsBack(t *testing.T) {
	reg := NewRegistry()
	reg.Register(stubProvider{id: "travelpayouts_data", caps: ProviderCapabilities{AirlineCoverageMode: model.CoverageUnknown}})
	leg := config.LegConfig{From: "MOW", To: "DXB", PreferredAirlines: []string{"SU"}, ProviderHint: "aviasales"}
	enabled := []config.ProviderConfig{{ID: "travelpayouts_data", Enabled: true}, {ID: "aviasales", Enabled: false}}
	p, ok := SelectProvider(leg, enabled, reg, config.AirlinePreferenceAdvisory)
	if !ok || p.Name() != "travelpayouts_data" {
		t.Fatalf("expected fallback to travelpayouts_data, got %v ok=%v", p, ok)
	}
}

func TestSelectProvider_UnknownNotRejectedForSU(t *testing.T) {
	reg := NewRegistry()
	reg.Register(stubProvider{id: "travelpayouts_data", caps: ProviderCapabilities{AirlineCoverageMode: model.CoverageUnknown}})
	leg := config.LegConfig{From: "MOW", To: "DXB", PreferredAirlines: []string{"SU"}}
	enabled := []config.ProviderConfig{{ID: "travelpayouts_data", Enabled: true}}
	p, ok := SelectProvider(leg, enabled, reg, config.AirlinePreferenceAdvisory)
	if !ok || p == nil {
		t.Fatal("unknown coverage provider must serve SU-preferred leg in advisory mode")
	}
}

func TestSelectProvider_KnownWithoutMatchRejected(t *testing.T) {
	reg := NewRegistry()
	reg.Register(stubProvider{id: "kiwi", caps: ProviderCapabilities{
		SupportedAirlines: map[string]bool{"FZ": true}, AirlineCoverageMode: model.CoverageKnown,
	}})
	leg := config.LegConfig{From: "MOW", To: "DXB", PreferredAirlines: []string{"SU"}}
	enabled := []config.ProviderConfig{{ID: "kiwi", Enabled: true}}
	p, ok := SelectProvider(leg, enabled, reg, config.AirlinePreferenceStrict)
	if ok || p != nil {
		t.Fatal("known provider without matching airline should not be selected in strict mode")
	}
}

func TestSelectProvider_PartialNotRejected(t *testing.T) {
	reg := NewRegistry()
	reg.Register(stubProvider{id: "unknown", caps: ProviderCapabilities{AirlineCoverageMode: model.CoverageUnknown}})
	leg := config.LegConfig{From: "MOW", To: "KTM", PreferredAirlines: []string{"XX"}}
	enabled := []config.ProviderConfig{{ID: "unknown", Enabled: true}}
	p, ok := SelectProvider(leg, enabled, reg, config.AirlinePreferenceAdvisory)
	if !ok || p == nil {
		t.Fatal("unknown coverage provider should remain selectable")
	}
}

func TestWarnSUPreference(t *testing.T) {
	reg := NewRegistry()
	reg.Register(stubProvider{id: "travelpayouts_data", caps: ProviderCapabilities{AirlineCoverageMode: model.CoverageUnknown}})
	enabled := []config.ProviderConfig{{ID: "travelpayouts_data", Enabled: true}}
	if WarnSUPreference([]string{"SU"}, enabled, reg) {
		t.Fatal("should not warn when advisory/unknown provider is enabled")
	}

	reg2 := NewRegistry()
	reg2.Register(stubProvider{id: "kiwi", caps: ProviderCapabilities{
		SupportedAirlines: map[string]bool{"FZ": true}, AirlineCoverageMode: model.CoverageKnown,
	}})
	enabled2 := []config.ProviderConfig{{ID: "kiwi", Enabled: true}}
	if !WarnSUPreference([]string{"SU"}, enabled2, reg2) {
		t.Fatal("expected warn when only known non-SU provider enabled")
	}
}

package provider

import (
	"github.com/kneumoin/nepal/internal/config"
	"github.com/kneumoin/nepal/internal/model"
)

// SelectProvider picks a provider for a leg. provider_hint is a preference only;
// if the hinted provider is disabled, selection falls back to other enabled providers.
func SelectProvider(leg config.LegConfig, enabled []config.ProviderConfig, reg *Registry, prefMode config.AirlinePreferenceMode) (Provider, bool) {
	enabledMap := make(map[string]bool)
	for _, p := range enabled {
		if p.Enabled {
			enabledMap[p.ID] = true
		}
	}

	if leg.ProviderHint != "" && enabledMap[leg.ProviderHint] {
		if p, ok := reg.Get(leg.ProviderHint); ok {
			if airlineMatch(p.Capabilities(), leg.PreferredAirlines, prefMode) {
				return p, true
			}
		}
	}

	var fallback Provider
	for _, p := range reg.All() {
		if !enabledMap[p.Name()] {
			continue
		}
		caps := p.Capabilities()
		if airlineMatch(caps, leg.PreferredAirlines, prefMode) {
			return p, true
		}
		if caps.AirlineCoverageMode == model.CoveragePartial || caps.AirlineCoverageMode == model.CoverageUnknown {
			if fallback == nil {
				fallback = p
			}
		}
	}
	return fallback, fallback != nil
}

func airlineMatch(caps ProviderCapabilities, preferred []string, mode config.AirlinePreferenceMode) bool {
	if len(preferred) == 0 {
		return true
	}
	if mode == config.AirlinePreferenceAdvisory &&
		(caps.AirlineCoverageMode == model.CoverageUnknown || caps.AirlineCoverageMode == model.CoveragePartial) {
		return true
	}
	for _, al := range preferred {
		if caps.SupportedAirlines[al] {
			return true
		}
	}
	if caps.AirlineCoverageMode == model.CoverageUnknown || caps.AirlineCoverageMode == model.CoveragePartial {
		return mode == config.AirlinePreferenceAdvisory
	}
	return false
}

// WarnSUPreference returns true when SU is preferred but no enabled provider declares SU
// and all enabled providers have known coverage (informational only; not a hard block).
func WarnSUPreference(preferred []string, enabled []config.ProviderConfig, reg *Registry) bool {
	if !prefersSU(preferred) {
		return false
	}
	hasSU := false
	hasAdvisory := false
	for _, pc := range enabled {
		if !pc.Enabled {
			continue
		}
		p, ok := reg.Get(pc.ID)
		if !ok {
			continue
		}
		caps := p.Capabilities()
		if caps.DeclaresSU() {
			hasSU = true
		}
		if caps.AirlineCoverageMode == model.CoverageUnknown || caps.AirlineCoverageMode == model.CoveragePartial {
			hasAdvisory = true
		}
	}
	if hasSU || hasAdvisory {
		return false
	}
	return true
}

func prefersSU(preferred []string) bool {
	for _, al := range preferred {
		if al == "SU" {
			return true
		}
	}
	return false
}

package provider

import (
	"github.com/kneumoin/nepal/internal/config"
	"github.com/kneumoin/nepal/internal/model"
)

type LegContext struct {
	Leg           config.LegConfig
	EnabledIDs    map[string]bool
	Registry      *Registry
	ProviderHints map[string]bool
}

func SelectProvider(leg config.LegConfig, enabled []config.ProviderConfig, reg *Registry) (Provider, bool) {
	enabledMap := make(map[string]bool)
	for _, p := range enabled {
		if p.Enabled {
			enabledMap[p.ID] = true
		}
	}

	if leg.ProviderHint != "" {
		if !enabledMap[leg.ProviderHint] {
			return nil, false
		}
		p, ok := reg.Get(leg.ProviderHint)
		return p, ok
	}

	var fallback Provider
	for _, p := range reg.All() {
		if !enabledMap[p.Name()] {
			continue
		}
		caps := p.Capabilities()
		if matchesPreferred(caps, leg.PreferredAirlines) {
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

func matchesPreferred(caps ProviderCapabilities, preferred []string) bool {
	if len(preferred) == 0 {
		return true
	}
	for _, al := range preferred {
		if caps.SupportedAirlines[al] {
			return true
		}
	}
	return false
}

func SUProviderAvailable(preferred []string, enabled []config.ProviderConfig, reg *Registry) bool {
	hasSU := false
	for _, al := range preferred {
		if al == "SU" {
			hasSU = true
			break
		}
	}
	if !hasSU {
		return true
	}
	for _, pc := range enabled {
		if !pc.Enabled {
			continue
		}
		p, ok := reg.Get(pc.ID)
		if !ok {
			continue
		}
		if p.Capabilities().DeclaresSU() {
			return true
		}
	}
	return false
}

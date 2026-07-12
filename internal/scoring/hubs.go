package scoring

import (
	"github.com/kneumoin/nepal/internal/config"
	"github.com/kneumoin/nepal/internal/model"
)

// IntermediateHubsFromLegs returns transit airports between MOW and KTM.
func IntermediateHubsFromLegs(legs []config.LegConfig) []string {
	if len(legs) < 2 {
		return nil
	}
	out := make([]string, 0, len(legs)-1)
	for i := 0; i < len(legs)-1; i++ {
		out = append(out, legs[i].To)
	}
	return out
}

func HubFromLegs(legs []config.LegConfig) string {
	hubs := IntermediateHubsFromLegs(legs)
	if len(hubs) == 0 {
		return ""
	}
	return hubs[0]
}

func WorstVisaFromLegs(legs []config.LegConfig) VisaCategory {
	order := map[VisaCategory]int{
		VisaLow: 0, VisaMedium: 1, VisaUnknown: 2, VisaHigh: 3, VisaRequiresVisa: 4,
	}
	worst := VisaLow
	for _, hub := range IntermediateHubsFromLegs(legs) {
		cat := HubVisaCategory(hub)
		if order[cat] > order[worst] {
			worst = cat
		}
	}
	return worst
}

func WorstOperationalDisruptionFromLegs(legs []config.LegConfig, cfg config.OperationalDisruptionConfig) model.OperationalDisruptionRisk {
	order := map[model.OperationalDisruptionRisk]int{
		model.OperationalDisruptionLow: 0, model.OperationalDisruptionUnknown: 1,
		model.OperationalDisruptionElevated: 2, model.OperationalDisruptionHigh: 3,
	}
	worst := model.OperationalDisruptionLow
	for _, hub := range IntermediateHubsFromLegs(legs) {
		r := HubOperationalDisruptionRisk(hub, cfg)
		if order[r] > order[worst] {
			worst = r
		}
	}
	return worst
}

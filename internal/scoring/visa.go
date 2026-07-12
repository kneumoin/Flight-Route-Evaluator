package scoring

import (
	"github.com/kneumoin/nepal/internal/config"
	"github.com/kneumoin/nepal/internal/model"
)

// VisaCategory describes transit visa risk at a hub.
type VisaCategory string

const (
	VisaLow           VisaCategory = "LOW"
	VisaMedium        VisaCategory = "MEDIUM"
	VisaHigh          VisaCategory = "HIGH"
	VisaRequiresVisa  VisaCategory = "REQUIRES_VISA"
	VisaUnknown       VisaCategory = "UNKNOWN"
)

// ApprovedTransitHubs re-exports the config list for scoring/report callers.
var ApprovedTransitHubs = config.ApprovedTransitHubs

// IsApprovedHub reports whether code is on the approved transit hub list.
func IsApprovedHub(iata string) bool {
	return config.IsApprovedHub(iata)
}

// HubVisaCategory returns static transit visa risk for an approved hub.
func HubVisaCategory(hubIATA string) VisaCategory {
	if c, ok := hubVisaRules[hubIATA]; ok {
		return c
	}
	if IsApprovedHub(hubIATA) {
		return VisaUnknown
	}
	return VisaUnknown
}

func VisaCategoryToRisk(c VisaCategory) model.Risk {
	switch c {
	case VisaLow:
		return model.RiskLow
	case VisaMedium:
		return model.RiskMedium
	case VisaHigh, VisaRequiresVisa:
		return model.RiskHigh
	case VisaUnknown:
		return model.RiskMedium
	default:
		return model.RiskMedium
	}
}

func VisaScoreForCategory(c VisaCategory) float64 {
	switch c {
	case VisaLow:
		return 100
	case VisaMedium:
		return 70
	case VisaHigh:
		return 40
	case VisaRequiresVisa:
		return 15
	case VisaUnknown:
		return 50
	default:
		return 50
	}
}

// RequiresTransitVisa is kept for compatibility; no longer used for hard reject.
func RequiresTransitVisa(hubIATA string) bool {
	return HubVisaCategory(hubIATA) == VisaRequiresVisa
}

// hubVisaRules: static transit assessment for Russian passport / expedition context.
var hubVisaRules = map[string]VisaCategory{
	"IST": VisaLow,
	"KWI": VisaLow,
	"DMM": VisaLow,
	"RUH": VisaLow,
	"DXB": VisaLow,
	"SHJ": VisaLow,
	"AUH": VisaLow,
	"DOH": VisaLow,
	"DEL": VisaRequiresVisa,
	"BOM": VisaRequiresVisa,
	"BLR": VisaRequiresVisa,
	"CMB": VisaMedium,
	"CCU": VisaRequiresVisa,
	"DAC": VisaHigh,
	"LXA": VisaRequiresVisa,
	"KMG": VisaMedium,
	"TFU": VisaMedium,
	"CAN": VisaMedium,
	"SZX": VisaMedium,
	"HKG": VisaLow,
	"BKK": VisaLow,
	"DMK": VisaLow,
	"KUL": VisaLow,
	"SIN": VisaLow,
	"ICN": VisaLow,
	"NRT": VisaLow,
	"TAS": VisaLow,
	"DYU": VisaMedium,
}

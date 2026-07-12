package scoring

import (
	"github.com/kneumoin/nepal/internal/config"
	"github.com/kneumoin/nepal/internal/model"
)

func HubOperationalDisruptionRisk(hubIATA string, cfg config.OperationalDisruptionConfig) model.OperationalDisruptionRisk {
	cfg.Normalize()
	return model.OperationalDisruptionRisk(cfg.HubRisk(hubIATA))
}

func OperationalDisruptionPenalty(level model.OperationalDisruptionRisk, cfg config.OperationalDisruptionConfig) float64 {
	if !cfg.Enabled {
		return 0
	}
	cfg.Normalize()
	return cfg.PenaltyFor(string(level))
}

package config

import (
	"fmt"
	"strings"
)

type RiskConfig struct {
	OperationalDisruption OperationalDisruptionConfig `yaml:"operational_disruption"`
}

type OperationalDisruptionConfig struct {
	Enabled      bool               `yaml:"enabled"`
	DefaultLevel string             `yaml:"default_level"`
	Penalties    map[string]float64 `yaml:"penalties"`
	Hubs         map[string]string  `yaml:"hubs"`
}

var validOperationalDisruptionLevels = map[string]bool{
	"LOW": true, "ELEVATED": true, "HIGH": true, "UNKNOWN": true,
}

var defaultOperationalDisruptionPenalties = map[string]float64{
	"LOW": 0, "ELEVATED": 5, "HIGH": 12, "UNKNOWN": 8,
}

func (c *Config) NormalizeRisk() {
	c.Risk.OperationalDisruption.normalize()
}

func (od *OperationalDisruptionConfig) Normalize() {
	od.normalize()
}

func (od *OperationalDisruptionConfig) normalize() {
	if od.DefaultLevel == "" {
		od.DefaultLevel = "LOW"
	}
	if od.Penalties == nil || len(od.Penalties) == 0 {
		od.Penalties = copyFloatMap(defaultOperationalDisruptionPenalties)
	}
}

func copyFloatMap(m map[string]float64) map[string]float64 {
	out := make(map[string]float64, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}

// HubRisk returns configured disruption level; unlisted hubs use default_level (LOW).
func (od OperationalDisruptionConfig) HubRisk(hubIATA string) string {
	hub := strings.ToUpper(strings.TrimSpace(hubIATA))
	if od.Hubs != nil {
		if level, ok := od.Hubs[hub]; ok {
			return strings.ToUpper(level)
		}
	}
	def := strings.ToUpper(strings.TrimSpace(od.DefaultLevel))
	if def == "" {
		def = "LOW"
	}
	return def
}

func (od OperationalDisruptionConfig) PenaltyFor(level string) float64 {
	level = strings.ToUpper(strings.TrimSpace(level))
	if od.Penalties != nil {
		if p, ok := od.Penalties[level]; ok {
			return p
		}
	}
	if p, ok := defaultOperationalDisruptionPenalties[level]; ok {
		return p
	}
	return defaultOperationalDisruptionPenalties["UNKNOWN"]
}

func validateOperationalDisruption(od OperationalDisruptionConfig) error {
	def := strings.ToUpper(od.DefaultLevel)
	if def != "" && !validOperationalDisruptionLevels[def] {
		return fmt.Errorf("risk.operational_disruption.default_level: invalid %s", od.DefaultLevel)
	}
	for hub, level := range od.Hubs {
		if !validIATA(hub) {
			return fmt.Errorf("risk.operational_disruption.hubs: invalid IATA %s", hub)
		}
		lvl := strings.ToUpper(level)
		if !validOperationalDisruptionLevels[lvl] {
			return fmt.Errorf("risk.operational_disruption.hubs.%s: invalid level %s", hub, level)
		}
	}
	for level := range od.Penalties {
		if !validOperationalDisruptionLevels[strings.ToUpper(level)] {
			return fmt.Errorf("risk.operational_disruption.penalties: invalid level %s", level)
		}
	}
	return nil
}

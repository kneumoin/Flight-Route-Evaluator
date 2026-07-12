package config_test

import (
	"path/filepath"
	"testing"

	"github.com/kneumoin/nepal/internal/config"
	"github.com/kneumoin/nepal/internal/model"
	"github.com/kneumoin/nepal/internal/scoring"
)

func TestOperationalDisruption_ConfigOverride(t *testing.T) {
	cfg, err := config.Load(filepath.Join("..", "..", "configs", "routes.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if scoring.HubOperationalDisruptionRisk("DXB", cfg.Risk.OperationalDisruption) != model.OperationalDisruptionHigh {
		t.Fatal("DXB expected HIGH from config")
	}
	if scoring.HubOperationalDisruptionRisk("IST", cfg.Risk.OperationalDisruption) != model.OperationalDisruptionLow {
		t.Fatal("IST should default LOW")
	}
	cfg.Risk.OperationalDisruption.Hubs["DXB"] = "ELEVATED"
	if scoring.HubOperationalDisruptionRisk("DXB", cfg.Risk.OperationalDisruption) != model.OperationalDisruptionElevated {
		t.Fatal("override failed")
	}
}

func TestOperationalDisruption_DefaultPenalties(t *testing.T) {
	od := config.OperationalDisruptionConfig{Enabled: true}
	if od.PenaltyFor("HIGH") != 12 {
		t.Fatalf("default HIGH penalty")
	}
}

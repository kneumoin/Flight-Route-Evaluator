package model_test

import (
	"strings"
	"testing"

	"github.com/kneumoin/nepal/internal/model"
)

func TestPriceComparison_MissingValuesNA(t *testing.T) {
	pc := &model.PriceComparison{
		PriceTarget:     &model.Money{Amount: 100000, Currency: "USD"},
		PricePlus1:      &model.Money{Amount: 99000, Currency: "USD"},
		PriceWindowMin:  &model.Money{Amount: 95000, Currency: "USD"},
		PriceWindowDays: 14,
	}
	s := pc.FormatPriceComparisonCompact("en")
	if !strings.Contains(s, "D-1 n/a") {
		t.Fatalf("expected D-1 n/a, got %q", s)
	}
	if !strings.Contains(s, "$1000") || !strings.Contains(s, "14d min $950") {
		t.Fatalf("unexpected format: %q", s)
	}
}

func TestPriceComparison_RULocale(t *testing.T) {
	pc := &model.PriceComparison{
		PriceTarget:     &model.Money{Amount: 100000, Currency: "USD"},
		PriceMinus1:     &model.Money{Amount: 108000, Currency: "USD"},
		PricePlus1:      &model.Money{Amount: 99000, Currency: "USD"},
		PriceWindowMin:  &model.Money{Amount: 95000, Currency: "USD"},
		PriceWindowDays: 14,
	}
	s := pc.FormatPriceComparisonCompact("ru")
	if !strings.Contains(s, "мин. 14д") {
		t.Fatalf("expected RU window label, got %q", s)
	}
}

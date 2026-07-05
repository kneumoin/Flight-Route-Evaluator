package scoring

import (
	"fmt"

	"github.com/kneumoin/nepal/internal/model"
)

// Rates to USD (major units); amount stored in minor units.
var ratesToUSD = map[string]float64{
	"USD": 1.0,
	"EUR": 1.08,
	"RUB": 0.011,
	"AED": 0.27,
	"QAR": 0.27,
	"INR": 0.012,
	"CNY": 0.14,
	"NPR": 0.0075,
}

func NormalizeMoney(m model.Money, target string) (*model.Money, error) {
	if m.Currency == target {
		cp := m
		return &cp, nil
	}
	rateFrom, ok1 := ratesToUSD[m.Currency]
	rateTo, ok2 := ratesToUSD[target]
	if !ok1 || !ok2 {
		return nil, fmt.Errorf("cannot convert %s to %s", m.Currency, target)
	}
	usdMajor := float64(m.Amount) / 100.0 * rateFrom
	targetMajor := usdMajor / rateTo
	return &model.Money{
		Amount:   int64(targetMajor * 100),
		Currency: target,
	}, nil
}

func NormalizeOfferPrice(o *model.Offer, target string) error {
	n, err := NormalizeMoney(o.Price, target)
	if err != nil {
		o.PriceNormalized = nil
		return err
	}
	o.PriceNormalized = n
	return nil
}

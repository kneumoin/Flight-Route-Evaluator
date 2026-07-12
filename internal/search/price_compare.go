package search

import (
	"context"

	"github.com/kneumoin/nepal/internal/config"
	"github.com/kneumoin/nepal/internal/model"
	"github.com/kneumoin/nepal/internal/provider"
	"github.com/kneumoin/nepal/internal/scoring"
)

const priceWindowDays = 14

func addDaysStr(date string, days int) string {
	return addDays(date, days)
}

// computePriceComparison calculates D-1/D/D+1 and 14-day window route prices.
func (e *Evaluator) computePriceComparison(ctx context.Context, branch config.BranchConfig, targetOffer *model.Offer) *model.PriceComparison {
	if targetOffer == nil || targetOffer.PriceNormalized == nil {
		return e.scanPriceWindow(ctx, branch)
	}
	tripDate := e.Config.Trip.DepartureDate
	cur := e.Config.Scoring.Currency
	pc := &model.PriceComparison{
		PriceTarget:         copyMoneyPtr(targetOffer.PriceNormalized),
		PriceTargetCurrency: cur,
		PriceWindowDays:     priceWindowDays,
	}
	pc.PriceMinus1 = e.routePriceOnDate(ctx, branch, addDaysStr(tripDate, -1), cur)
	pc.PricePlus1 = e.routePriceOnDate(ctx, branch, addDaysStr(tripDate, 1), cur)
	minAmt, minDate := e.minRoutePriceInWindow(ctx, branch, tripDate, priceWindowDays, cur)
	if minAmt != nil {
		pc.PriceWindowMin = minAmt
		pc.PriceWindowMinDate = minDate
	}
	return pc
}

func (e *Evaluator) scanPriceWindow(ctx context.Context, branch config.BranchConfig) *model.PriceComparison {
	tripDate := e.Config.Trip.DepartureDate
	cur := e.Config.Scoring.Currency
	pc := &model.PriceComparison{PriceWindowDays: priceWindowDays}
	pc.PriceMinus1 = e.routePriceOnDate(ctx, branch, addDaysStr(tripDate, -1), cur)
	pc.PriceTarget = e.routePriceOnDate(ctx, branch, tripDate, cur)
	pc.PricePlus1 = e.routePriceOnDate(ctx, branch, addDaysStr(tripDate, 1), cur)
	minAmt, minDate := e.minRoutePriceInWindow(ctx, branch, tripDate, priceWindowDays, cur)
	if minAmt != nil {
		pc.PriceWindowMin = minAmt
		pc.PriceWindowMinDate = minDate
	}
	if pc.PriceTarget != nil {
		pc.PriceTargetCurrency = cur
	}
	return pc
}

func (e *Evaluator) routePriceOnDate(ctx context.Context, branch config.BranchConfig, leg1Date, currency string) *model.Money {
	legOffers := make([][]model.Offer, len(branch.Legs))
	prefMode := e.Config.Constraints.AirlinePreferenceMode
	window := e.Config.Coverage.WindowDays

	for i, leg := range branch.Legs {
		p, ok := providerForLeg(e, leg, prefMode)
		if !ok {
			return nil
		}
		targetDate := leg1Date
		notBefore := ""
		var prevOffers []model.Offer
		if i > 0 {
			prevOffers = legOffers[i-1]
			targetDate = legTargetDate(prevOffers, leg1Date)
			notBefore = earliestLeg2CalendarDate(prevOffers, branch.MinConnectionHours)
		}
		searchDates := legSearchDates(i, leg1Date, prevOffers, branch, e.Config.OutboundFirstLegDates())
		var allOffers []model.Offer
		for _, d := range searchDates {
			q := baseQuery(e, leg, d, window)
			q.TargetDate = targetDate
			q.NotBeforeDate = notBefore
			offers, err := p.Search(ctx, q)
			if err != nil {
				return nil
			}
			allOffers = append(allOffers, tagOffersForLeg(offers, d, targetDate)...)
		}
		if len(allOffers) == 0 {
			fallbackQ := baseQuery(e, leg, targetDate, window)
			fallbackQ.AllowCoverageFallback = true
			fallbackQ.TargetDate = targetDate
			fallbackQ.NotBeforeDate = notBefore
			offers, err := p.Search(ctx, fallbackQ)
			if err != nil {
				return nil
			}
			allOffers = append(allOffers, offers...)
		}
		if i > 0 {
			allOffers = filterLeg2AfterEarliest(allOffers, prevOffers, branch.MinConnectionHours)
		}
		if len(allOffers) == 0 {
			return nil
		}
		legOffers[i] = allOffers
	}
	combined, ok := combineLegOffers(branch, legOffers, currency)
	if !ok || combined.PriceNormalized == nil {
		return nil
	}
	m := *combined.PriceNormalized
	return &m
}

func (e *Evaluator) minRoutePriceInWindow(ctx context.Context, branch config.BranchConfig, center string, windowDays int, currency string) (*model.Money, string) {
	var best *model.Money
	var bestDate string
	for d := -windowDays; d <= windowDays; d++ {
		date := addDaysStr(center, d)
		p := e.routePriceOnDate(ctx, branch, date, currency)
		if p == nil {
			continue
		}
		if best == nil || p.Amount < best.Amount {
			cp := *p
			best = &cp
			bestDate = date
		}
	}
	return best, bestDate
}

func providerForLeg(e *Evaluator, leg config.LegConfig, prefMode config.AirlinePreferenceMode) (provider.Provider, bool) {
	if e.MockOnly {
		return e.Registry.Get("mock")
	}
	p, ok := provider.SelectProvider(leg, e.Config.Providers, e.Registry, prefMode)
	return p, ok && p != nil
}

func copyMoneyPtr(m *model.Money) *model.Money {
	if m == nil {
		return nil
	}
	cp := *m
	return &cp
}

func hubVisaCategoryForBranch(branch config.BranchConfig) model.VisaCategory {
	return model.VisaCategory(scoring.WorstVisaFromLegs(branch.Legs))
}

func applyVisaToOffer(offer *model.Offer, branch config.BranchConfig) {
	if offer == nil {
		return
	}
	cat := scoring.WorstVisaFromLegs(branch.Legs)
	offer.VisaRisk = scoring.VisaCategoryToRisk(cat)
}

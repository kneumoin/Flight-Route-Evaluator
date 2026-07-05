package search

import (
	"context"
	"fmt"
	"log"
	"sort"
	"time"

	"github.com/kneumoin/nepal/internal/config"
	"github.com/kneumoin/nepal/internal/model"
	"github.com/kneumoin/nepal/internal/provider"
	"github.com/kneumoin/nepal/internal/scoring"
)

type Evaluator struct {
	Config   *config.Config
	Registry *provider.Registry
	Verbose  bool
	MockOnly bool
}

func (e *Evaluator) Evaluate(ctx context.Context) (*model.EvaluationResult, error) {
	results := make([]model.BranchResult, 0, len(e.Config.Branches))

	for _, branch := range e.Config.Branches {
		br, err := e.evaluateBranch(ctx, branch)
		if err != nil {
			return nil, err
		}
		results = append(results, br)
	}

	sort.SliceStable(results, func(i, j int) bool {
		si, sj := scoreOf(results[i]), scoreOf(results[j])
		if si != sj {
			return si > sj
		}
		return results[i].BranchID < results[j].BranchID
	})

	return &model.EvaluationResult{
		GeneratedAt: timeNow(),
		Trip: model.TripMeta{
			Origin:        e.Config.Trip.Origin,
			Destination:   e.Config.Trip.Destination,
			DepartureDate: e.Config.Trip.DepartureDate,
			Passengers:    e.Config.Trip.Passengers,
			Cabin:         e.Config.Trip.Cabin,
		},
		Branches: results,
	}, nil
}

func scoreOf(br model.BranchResult) float64 {
	if br.Score == nil {
		return -1
	}
	return *br.Score
}

func (e *Evaluator) evaluateBranch(ctx context.Context, branch config.BranchConfig) (model.BranchResult, error) {
	res := model.BranchResult{BranchID: branch.ID, BranchName: branch.Name, Status: model.StatusUnavailable}

	for _, leg := range branch.Legs {
		if !provider.SUProviderAvailable(leg.PreferredAirlines, e.Config.Providers, e.Registry) {
			if e.Verbose {
				log.Printf("warn: no SU provider for leg %s-%s", leg.From, leg.To)
			}
			res.ReasonCodes = []model.ReasonCode{model.ReasonNoProvider}
			return res, nil
		}
	}

	hub := scoring.HubFromLegs(branch.Legs)
	if scoring.RequiresTransitVisa(hub) && e.Config.Constraints.AvoidTransferVisas {
		res.Status = model.StatusRejected
		res.ReasonCodes = []model.ReasonCode{model.ReasonTransitVisaRequired}
		return res, nil
	}

	legOffers := make([][]model.Offer, len(branch.Legs))
	providersUsed := map[string]bool{}

	for i, leg := range branch.Legs {
		p, ok := provider.SelectProvider(leg, e.Config.Providers, e.Registry)
		if e.MockOnly {
			p, ok = e.Registry.Get("mock")
		}
		if !ok || p == nil {
			res.ReasonCodes = []model.ReasonCode{model.ReasonNoProvider}
			return res, nil
		}
		q := model.Query{
			From: leg.From, To: leg.To, Date: e.Config.Trip.DepartureDate,
			Passengers: e.Config.Trip.Passengers, Cabin: e.Config.Trip.Cabin,
			Airlines: leg.PreferredAirlines,
		}
		offers, err := p.Search(ctx, q)
		if err != nil {
			res.ReasonCodes = []model.ReasonCode{model.ReasonAPIError}
			return res, nil
		}
		if len(offers) == 0 {
			res.ReasonCodes = []model.ReasonCode{model.ReasonNoOffers}
			return res, nil
		}
		legOffers[i] = offers
		providersUsed[p.Name()] = true
	}

	combined, ok := combineLegOffers(branch, legOffers, e.Config.Scoring.Currency)
	if !ok {
		res.ReasonCodes = []model.ReasonCode{model.ReasonNoOffers}
		return res, nil
	}

	combined.BranchID = branch.ID
	combined.SelfTransfer = branch.Type == "self_transfer"

	connH := combined.ConnectionDuration.Hours()
	if codes := scoring.CheckConnection(connH, branch); len(codes) > 0 {
		res.Status = model.StatusRejected
		res.ReasonCodes = codes
		return res, nil
	}

	if reject, codes := scoring.CheckBaggage(combined.CheckedBaggageKg, e.Config.Constraints); reject {
		res.Status = model.StatusRejected
		res.ReasonCodes = codes
		return res, nil
	}

	if err := scoring.NormalizeOfferPrice(&combined, e.Config.Scoring.Currency); err != nil {
		res.ReasonCodes = []model.ReasonCode{model.ReasonCurrencyUnconvertible}
		return res, nil
	}

	if combined.PriceNormalized == nil {
		res.ReasonCodes = []model.ReasonCode{model.ReasonCurrencyUnconvertible}
		return res, nil
	}

	score, bd := scoring.ScoreOffer(combined, branch, e.Config.Scoring, e.Config.Constraints)
	res.Status = model.StatusOK
	res.Offer = &combined
	res.Score = &score
	res.Breakdown = &bd
	for p := range providersUsed {
		res.Providers = append(res.Providers, p)
	}
	sort.Strings(res.Providers)
	return res, nil
}

func combineLegOffers(branch config.BranchConfig, legs [][]model.Offer, scoringCurrency string) (model.Offer, bool) {
	if len(legs) == 1 {
		o := legs[0][0]
		return o, true
	}
	best := model.Offer{}
	found := false
	var bestPrice int64

	for _, o1 := range legs[0] {
		for _, o2 := range legs[1] {
			if len(o1.Segments) == 0 || len(o2.Segments) == 0 {
				continue
			}
			conn := scoring.ConnectionDuration(o1.Segments[len(o1.Segments)-1].Arrival, o2.Segments[0].Departure)
			connH := conn.Hours()
			if connH < branch.MinConnectionHours || connH > branch.MaxConnectionHours {
				continue
			}
			combined := mergeOffers(o1, o2, conn)
			leg1n, e1 := scoring.NormalizeMoney(o1.Price, scoringCurrency)
			leg2n, e2 := scoring.NormalizeMoney(o2.Price, scoringCurrency)
			if e1 != nil || e2 != nil {
				continue
			}
			sum := model.Money{Amount: leg1n.Amount + leg2n.Amount, Currency: scoringCurrency}
			combined.PriceNormalized = &sum
			if o1.Price.Currency == o2.Price.Currency {
				combined.Price = model.Money{Amount: o1.Price.Amount + o2.Price.Amount, Currency: o1.Price.Currency}
			} else {
				combined.Price = o1.Price
			}
			if !found || combined.PriceNormalized.Amount < bestPrice {
				best = combined
				bestPrice = combined.PriceNormalized.Amount
				found = true
			}
		}
	}
	return best, found
}

func mergeOffers(o1, o2 model.Offer, conn time.Duration) model.Offer {
	segs := append([]model.Segment{}, o1.Segments...)
	segs = append(segs, o2.Segments...)
	price := model.Money{
		Amount:   o1.Price.Amount + o2.Price.Amount,
		Currency: o1.Price.Currency,
	}
	if o1.Price.Currency != o2.Price.Currency {
		// keep first currency; normalization happens later on combined if same - for mock same currencies per branch typically
		price.Currency = o1.Price.Currency
	}
	bag := minBaggage(o1.CheckedBaggageKg, o2.CheckedBaggageKg)
	return model.Offer{
		Provider:           fmt.Sprintf("%s+%s", o1.Provider, o2.Provider),
		Segments:           segs,
		Price:              price,
		TotalDuration:      o1.TotalDuration + conn + o2.TotalDuration,
		ConnectionDuration: conn,
		CheckedBaggageKg:   bag,
		VisaRisk:           maxRisk(o1.VisaRisk, o2.VisaRisk),
	}
}

func minBaggage(a, b *int) *int {
	if a == nil {
		return b
	}
	if b == nil {
		return a
	}
	v := *a
	if *b < v {
		v = *b
	}
	return &v
}

func maxRisk(a, b model.Risk) model.Risk {
	order := map[model.Risk]int{model.RiskLow: 0, model.RiskMedium: 1, model.RiskHigh: 2, model.RiskRejected: 3}
	if order[b] > order[a] {
		return b
	}
	return a
}

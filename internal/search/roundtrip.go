package search

import (
	"context"
	"fmt"
	"sort"

	"github.com/kneumoin/nepal/internal/config"
	"github.com/kneumoin/nepal/internal/model"
	"github.com/kneumoin/nepal/internal/scoring"
)

const (
	directionOutbound = "outbound"
	directionReturn   = "return"
)

type legEvalContext struct {
	tripDate      string
	firstLegDates []string
	direction     string
}

func reverseBranch(b config.BranchConfig) config.BranchConfig {
	return reverseBranchConfig(b)
}

// ReverseBranchForTest exposes branch reversal for unit tests.
func ReverseBranchForTest(b config.BranchConfig) config.BranchConfig {
	return reverseBranchConfig(b)
}

func reverseBranchConfig(b config.BranchConfig) config.BranchConfig {
	rb := b
	rb.Name = b.Name + " (return)"
	n := len(b.Legs)
	rb.Legs = make([]config.LegConfig, n)
	for i := 0; i < n; i++ {
		src := b.Legs[n-1-i]
		rb.Legs[i] = config.LegConfig{
			From:              src.To,
			To:                src.From,
			PreferredAirlines: append([]string(nil), src.PreferredAirlines...),
			ProviderHint:      src.ProviderHint,
		}
	}
	return rb
}

func (e *Evaluator) mergeRoundTripBranch(
	outbound, ret model.BranchResult,
	branch config.BranchConfig,
	outCov, retCov []model.RouteCoverageRow,
) (model.BranchResult, []model.RouteCoverageRow) {
	coverage := append(append([]model.RouteCoverageRow{}, outCov...), retCov...)

	if outbound.Status == model.StatusUnavailable && ret.Status == model.StatusUnavailable {
		outbound.ReasonCodes = appendReasonCode(outbound.ReasonCodes, model.ReasonNoRouteData)
		return outbound, coverage
	}

	res := outbound
	res.OutboundOffer = outbound.Offer
	res.ReturnOffer = ret.Offer

	if outbound.Offer == nil && ret.Offer == nil {
		res.Status = model.StatusPartial
		res.ReasonCodes = appendReasonCode(res.ReasonCodes, model.ReasonPartialData)
		return res, coverage
	}

	if outbound.Offer == nil || ret.Offer == nil {
		res.Status = model.StatusPartial
		res.ReasonCodes = appendReasonCode(res.ReasonCodes, model.ReasonPartialData)
		if res.Offer == nil {
			res.Offer = firstNonNilOffer(outbound.Offer, ret.Offer)
		}
		return res, coverage
	}

	combined := mergeDirectionOffers(*outbound.Offer, *ret.Offer, branch, e.Config.Trip.DepartureDate, e.Config.Trip.ReturnDate)
	combined.BranchID = branch.ID
	applyVisaToOffer(&combined, branch)

	if err := scoring.NormalizeOfferPrice(&combined, e.Config.Scoring.Currency); err != nil || combined.PriceNormalized == nil {
		res.Status = model.StatusPartial
		res.ReasonCodes = appendReasonCode(res.ReasonCodes, model.ReasonPartialData)
		res.Offer = &combined
		return res, coverage
	}

	allLegs := append(append([]config.LegConfig{}, branch.Legs...), reverseBranch(branch).Legs...)
	visa := scoring.WorstVisaFromLegs(allLegs)
	disruption := scoring.WorstOperationalDisruptionFromLegs(allLegs, e.Config.Risk.OperationalDisruption)

	if branchUsesPartialData(combined) || outbound.Status == model.StatusPartial || ret.Status == model.StatusPartial {
		res.Status = model.StatusPartial
		res.ReasonCodes = appendReasonCode(res.ReasonCodes, model.ReasonPartialData)
	} else {
		res.Status = model.StatusOK
	}

	score, bd := scoring.ScoreOffer(combined, branch, e.Config.Scoring, e.Config.Constraints, visa, disruption, e.Config.Risk.OperationalDisruption)
	res.Offer = &combined
	res.Score = &score
	res.Breakdown = &bd
	res.OperationalDisruptionPenalty = bd.RegionalDisruptionPenalty
	res.PriceComparison = e.computePriceComparison(context.Background(), branch, &combined)

	res.Providers = uniqueStrings(append(res.Providers, ret.Providers...))
	sort.Strings(res.Providers)
	return res, coverage
}

func mergeDirectionOffers(outbound, ret model.Offer, branch config.BranchConfig, outboundDate, returnDate string) model.Offer {
	price := model.Money{
		Amount:   outbound.Price.Amount + ret.Price.Amount,
		Currency: outbound.Price.Currency,
	}
	if outbound.Price.Currency != ret.Price.Currency && ret.Price.Currency != "" {
		price.Currency = outbound.Price.Currency
	}

	notes := mergeNotes(outbound.Notes, ret.Notes)
	if outboundDate != "" && returnDate != "" {
		notes = appendNote(notes, fmt.Sprintf("Round trip: outbound %s, return %s", outboundDate, returnDate))
	}

	return model.Offer{
		Provider:           fmt.Sprintf("%s+%s", outbound.Provider, ret.Provider),
		Segments:           append(append([]model.Segment{}, outbound.Segments...), ret.Segments...),
		Price:              price,
		TotalDuration:      outbound.TotalDuration + ret.TotalDuration,
		ConnectionDuration: outbound.ConnectionDuration + ret.ConnectionDuration,
		CheckedBaggageKg:   minBaggage(outbound.CheckedBaggageKg, ret.CheckedBaggageKg),
		VisaRisk:           maxRisk(outbound.VisaRisk, ret.VisaRisk),
		DataQuality:        mergeDataQuality(outbound.DataQuality, ret.DataQuality),
		Notes:              notes,
		TimingVerified:     outbound.TimingVerified && ret.TimingVerified,
		ConnectionVerified: outbound.ConnectionVerified && ret.ConnectionVerified,
		EstimatedDate:      outbound.EstimatedDate || ret.EstimatedDate,
		LegDetails:         append(append([]model.LegDetail{}, outbound.LegDetails...), ret.LegDetails...),
		SelfTransfer:       branch.Type == "self_transfer",
	}
}

func firstNonNilOffer(offers ...*model.Offer) *model.Offer {
	for _, o := range offers {
		if o != nil {
			return o
		}
	}
	return nil
}

func uniqueStrings(in []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(in))
	for _, s := range in {
		if s == "" || seen[s] {
			continue
		}
		seen[s] = true
		out = append(out, s)
	}
	return out
}

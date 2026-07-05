package scoring

import (
	"strconv"
	"strings"
	"time"

	"github.com/kneumoin/nepal/internal/config"
	"github.com/kneumoin/nepal/internal/model"
)

var visaRequiredHubs = map[string]bool{
	"DEL": true,
}

func RequiresTransitVisa(hubIATA string) bool {
	return visaRequiredHubs[hubIATA]
}

func HubFromLegs(legs []config.LegConfig) string {
	if len(legs) < 2 {
		return ""
	}
	return legs[0].To
}

func CheckConnection(hours float64, branch config.BranchConfig) []model.ReasonCode {
	if hours < branch.MinConnectionHours {
		return []model.ReasonCode{model.ReasonConnectionTooShort}
	}
	if hours > branch.MaxConnectionHours {
		return []model.ReasonCode{model.ReasonConnectionTooLong}
	}
	return nil
}

func CheckBaggage(kg *int, c config.ConstraintsConfig) (reject bool, codes []model.ReasonCode) {
	if !c.Baggage.CheckedRequired {
		return false, nil
	}
	if kg == nil {
		return true, []model.ReasonCode{model.ReasonBaggageUnknown}
	}
	if *kg < c.Baggage.MinCheckedKg {
		return false, nil // penalty only in scoring
	}
	return false, nil
}

func ScoreOffer(offer model.Offer, branch config.BranchConfig, sc config.ScoringConfig, c config.ConstraintsConfig) (float64, model.ScoreBreakdown) {
	w := sc.Weights
	if w.Price == 0 {
		w = config.ScoringWeights{Price: 1, Duration: 0.5, Baggage: 0.3, Visa: 1, SelfTransfer: 0.8, LateArrival: 0.4}
	}

	priceNorm := float64(offer.PriceNormalized.Amount) / 100.0
	priceScore := clamp(100 - priceNorm/20) // lower price -> higher score

	durH := offer.TotalDuration.Hours()
	durScore := clamp(100 - durH*2)

	bagScore := 80.0
	if c.Baggage.CheckedRequired {
		if offer.CheckedBaggageKg == nil {
			bagScore = 0
		} else if *offer.CheckedBaggageKg < c.Baggage.MinCheckedKg {
			bagScore = 30
		} else {
			bagScore = 100
		}
	}

	visaScore := 100.0
	if offer.VisaRisk == model.RiskHigh {
		visaScore = 40
	} else if offer.VisaRisk == model.RiskMedium {
		visaScore = 70
	}

	selfScore := 100.0
	if offer.SelfTransfer || branch.Type == "self_transfer" {
		selfScore = 60
		connH := offer.ConnectionDuration.Hours()
		if connH < branch.MinConnectionHours {
			selfScore = 20
		}
	}

	lateScore := 100.0
	if sc.LateArrivalAfter != "" && len(offer.Segments) > 0 {
		arr := offer.Segments[len(offer.Segments)-1].Arrival
		cutoff := parseCutoff(sc.LateArrivalAfter, arr)
		if arr.After(cutoff) {
			lateScore = 50
		}
	}

	total := (priceScore*w.Price + durScore*w.Duration + bagScore*w.Baggage +
		visaScore*w.Visa + selfScore*w.SelfTransfer + lateScore*w.LateArrival) /
		(w.Price + w.Duration + w.Baggage + w.Visa + w.SelfTransfer + w.LateArrival)

	bd := model.ScoreBreakdown{
		Price: priceScore, Duration: durScore, Baggage: bagScore,
		Visa: visaScore, SelfTransfer: selfScore, LateArrival: lateScore,
		Total: total,
	}
	return clamp(total), bd
}

func parseCutoff(hhmm string, ref time.Time) time.Time {
	parts := strings.Split(hhmm, ":")
	h, m := 18, 0
	if len(parts) >= 1 {
		h, _ = strconv.Atoi(parts[0])
	}
	if len(parts) >= 2 {
		m, _ = strconv.Atoi(parts[1])
	}
	return time.Date(ref.Year(), ref.Month(), ref.Day(), h, m, 0, 0, ref.Location())
}

func clamp(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 100 {
		return 100
	}
	return v
}

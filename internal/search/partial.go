package search

import (
	"context"
	"sort"
	"time"

	"github.com/kneumoin/nepal/internal/config"
	"github.com/kneumoin/nepal/internal/model"
	"github.com/kneumoin/nepal/internal/scoring"
)

func coverageRowHasInfo(row model.RouteCoverageRow) bool {
	return row.CoverageDays > 0 ||
		row.MinPrice != nil ||
		row.CheapestDateNearTarget != "" ||
		row.Airline != "" ||
		len(row.AvailableAirlines) > 0
}

func coverageRowForLeg(rows []model.RouteCoverageRow, legIndex int, direction string) (model.RouteCoverageRow, bool) {
	for _, r := range rows {
		if r.LegIndex == legIndex && (direction == "" || r.Direction == direction) {
			return r, true
		}
	}
	return model.RouteCoverageRow{}, false
}

func branchHasAnyRouteInfo(coverageRows []model.RouteCoverageRow, legOffers [][]model.Offer) bool {
	for _, row := range coverageRows {
		if coverageRowHasInfo(row) {
			return true
		}
	}
	for _, leg := range legOffers {
		if len(leg) > 0 {
			return true
		}
	}
	return false
}

func offersFromCoverage(row model.RouteCoverageRow, from, to, targetDate, currency string) []model.Offer {
	if !coverageRowHasInfo(row) {
		return nil
	}
	searchDate := row.SelectedDate
	if searchDate == "" {
		searchDate = row.CheapestDateNearTarget
	}
	if searchDate == "" {
		searchDate = targetDate
	}
	estimated := row.EstimatedDate || searchDate != targetDate

	notes := []string{model.NoteRouteDataIncomplete}
	if row.MinPrice == nil {
		notes = append(notes, model.NoteNoPriceOnTarget)
	} else if estimated {
		notes = append(notes, model.NotePriceFromNearbyDate)
	}

	price := model.Money{Currency: currency}
	if row.MinPrice != nil {
		price = *row.MinPrice
	}

	airlines := append([]string(nil), row.AvailableAirlines...)
	if row.Airline != "" {
		airlines = model.MergeAirlineLists(airlines, []string{row.Airline})
	}

	seg := syntheticSegment(from, to, searchDate, row.Airline)
	return []model.Offer{{
		Provider:          "travelpayouts_data",
		Segments:          []model.Segment{seg},
		Price:             price,
		TotalDuration:     seg.Duration,
		DataQuality:       model.DataQualityCached,
		Notes:             notes,
		SearchDate:        searchDate,
		EstimatedDate:     estimated,
		TimingVerified:    false,
		AvailableAirlines: airlines,
		Transfers:         row.Transfers,
	}}
}

func syntheticSegment(from, to, date, airline string) model.Segment {
	fromLoc, _ := scoring.AirportLocation(from)
	toLoc, _ := scoring.AirportLocation(to)
	day, _ := time.Parse("2006-01-02", date)
	sched := defaultLegSchedule(from, to)
	dep := time.Date(day.Year(), day.Month(), day.Day(), sched.depHour, 0, 0, 0, fromLoc)
	dur := time.Duration(sched.durH) * time.Hour
	arr := dep.Add(dur).In(toLoc)
	return model.Segment{
		From: from, To: to,
		Departure: dep, Arrival: arr,
		Airline: airline, Duration: dur,
	}
}

type legSched struct{ depHour, durH int }

func defaultLegSchedule(from, to string) legSched {
	key := from + "-" + to
	if s, ok := defaultLegSchedules[key]; ok {
		return s
	}
	revKey := to + "-" + from
	if s, ok := defaultLegSchedules[revKey]; ok {
		return s
	}
	return legSched{depHour: 8, durH: 5}
}

var defaultLegSchedules = map[string]legSched{
	"MOW-DOH": {8, 5}, "DOH-KTM": {16, 4},
	"MOW-DXB": {9, 5}, "DXB-KTM": {23, 4},
	"MOW-DEL": {7, 6}, "DEL-KTM": {16, 2},
	"MOW-IST": {10, 4}, "IST-KTM": {18, 6},
	"MOW-TFU": {6, 8}, "TFU-KTM": {20, 3},
	"MOW-TAS": {8, 4}, "TAS-DEL": {10, 3},
	"MOW-DYU": {8, 4}, "DYU-DEL": {10, 3},
}

func branchUsesPartialData(o model.Offer) bool {
	if o.EstimatedDate {
		return true
	}
	for _, leg := range o.LegDetails {
		if leg.EstimatedDate {
			return true
		}
	}
	for _, n := range o.Notes {
		switch n {
		case model.NoteNoPriceOnTarget, model.NotePriceFromNearbyDate, model.NoteRouteDataIncomplete:
			return true
		}
	}
	return false
}

func appendReasonCode(codes []model.ReasonCode, code model.ReasonCode) []model.ReasonCode {
	for _, c := range codes {
		if c == code {
			return codes
		}
	}
	return append(codes, code)
}

func (e *Evaluator) finishPartialBranch(
	ctx context.Context,
	res model.BranchResult,
	branch config.BranchConfig,
	coverageRows []model.RouteCoverageRow,
	legOffers [][]model.Offer,
	providersUsed map[string]bool,
	disruption model.OperationalDisruptionRisk,
	tripDate string,
	direction string,
) (model.BranchResult, []model.RouteCoverageRow, error) {
	if !branchHasAnyRouteInfo(coverageRows, legOffers) {
		res.ReasonCodes = []model.ReasonCode{model.ReasonNoRouteData}
		return res, coverageRows, nil
	}

	res.Status = model.StatusPartial
	res.ReasonCodes = appendReasonCode(res.ReasonCodes, model.ReasonPartialData)

	cur := e.Config.Scoring.Currency
	if cur == "" {
		cur = "USD"
	}
	for i, leg := range branch.Legs {
		if len(legOffers[i]) > 0 {
			continue
		}
		row, ok := coverageRowForLeg(coverageRows, i, direction)
		if !ok {
			continue
		}
		prev := legOffers[i-1]
		if i == 0 {
			prev = nil
		}
		targetDate := tripDate
		if i > 0 {
			targetDate = legTargetDate(prev, tripDate)
		}
		if cov := offersFromCoverage(row, leg.From, leg.To, targetDate, cur); len(cov) > 0 {
			legOffers[i] = cov
		}
	}

	res.LegAirlines = nil
	for i, leg := range branch.Legs {
		targetDate := tripDate
		if i > 0 {
			targetDate = legTargetDate(legOffers[i-1], tripDate)
		}
		airlines := collectLegAirlines(legOffers[i], targetDate)
		if row, ok := coverageRowForLeg(coverageRows, i, direction); ok {
			if row.Airline != "" {
				airlines = model.MergeAirlineLists(airlines, []string{row.Airline})
			}
			airlines = model.MergeAirlineLists(airlines, row.AvailableAirlines)
		}
		res.LegAirlines = append(res.LegAirlines, model.LegAirlines{
			From:              leg.From,
			To:                leg.To,
			TargetDate:        targetDate,
			AvailableAirlines: airlines,
		})
	}

	combined, ok := combineLegOffers(branch, legOffers, e.Config.Scoring.Currency)
	if ok {
		combined.BranchID = branch.ID
		combined.SelfTransfer = branch.Type == "self_transfer"
		applyVisaToOffer(&combined, branch)
		combined.Notes = appendNote(combined.Notes, model.NoteRouteDataIncomplete)
		if err := scoring.NormalizeOfferPrice(&combined, e.Config.Scoring.Currency); err == nil && combined.PriceNormalized != nil {
			score, bd := scoring.ScoreOffer(combined, branch, e.Config.Scoring, e.Config.Constraints, scoring.WorstVisaFromLegs(branch.Legs), disruption, e.Config.Risk.OperationalDisruption)
			res.Offer = &combined
			res.Score = &score
			res.Breakdown = &bd
			res.OperationalDisruptionPenalty = bd.RegionalDisruptionPenalty
			res.PriceComparison = e.computePriceComparison(ctx, branch, &combined)
		} else {
			res.PriceComparison = e.scanPriceWindow(ctx, branch)
		}
	} else {
		res.PriceComparison = e.scanPriceWindow(ctx, branch)
	}

	for p := range providersUsed {
		res.Providers = append(res.Providers, p)
	}
	sort.Strings(res.Providers)
	return res, coverageRows, nil
}

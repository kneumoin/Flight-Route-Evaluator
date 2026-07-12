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
	Progress bool
	MockOnly bool
}

func (e *Evaluator) Evaluate(ctx context.Context) (*model.EvaluationResult, error) {
	results := make([]model.BranchResult, 0, len(e.Config.Branches))
	var coverage []model.RouteCoverageRow

	total := len(e.Config.Branches)
	for i, branch := range e.Config.Branches {
		e.progressBranchStart(i+1, total, branch)
		branchStart := time.Now()
		br, branchCov, err := e.evaluateBranch(ctx, branch)
		if err != nil {
			return nil, err
		}
		e.progressBranchDone(br, time.Since(branchStart))
		results = append(results, br)
		coverage = append(coverage, branchCov...)
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
			Origin:              e.Config.Trip.Origin,
			Destination:         e.Config.Trip.Destination,
			DepartureDate:       e.Config.Trip.DepartureDate,
			ReturnDate:          e.Config.Trip.ReturnDate,
			ReturnDateEnd:       e.Config.Trip.ReturnDateEnd,
			OutboundForwardDays: e.Config.Coverage.OutboundForwardDays,
			Passengers:          e.Config.Trip.Passengers,
			Cabin:               e.Config.Trip.Cabin,
		},
		Branches:      results,
		RouteCoverage: coverage,
	}, nil
}

func scoreOf(br model.BranchResult) float64 {
	if br.Score == nil {
		return -1
	}
	return *br.Score
}

func (e *Evaluator) evaluateBranch(ctx context.Context, branch config.BranchConfig) (model.BranchResult, []model.RouteCoverageRow, error) {
	outboundCtx := legEvalContext{
		tripDate:      e.Config.Trip.DepartureDate,
		firstLegDates: e.Config.OutboundFirstLegDates(),
		direction:     directionOutbound,
	}
	out, outCov, err := e.evaluateDirection(ctx, branch, outboundCtx)
	if err != nil {
		return out, outCov, err
	}
	if !e.Config.RoundTrip() {
		return out, outCov, nil
	}

	retBranch := reverseBranch(branch)
	returnCtx := legEvalContext{
		tripDate:      e.Config.Trip.ReturnDate,
		firstLegDates: e.Config.ReturnFirstLegDates(),
		direction:     directionReturn,
	}
	ret, retCov, err := e.evaluateDirection(ctx, retBranch, returnCtx)
	if err != nil {
		return out, outCov, err
	}
	merged, mergedCov := e.mergeRoundTripBranch(out, ret, branch, outCov, retCov)
	return merged, mergedCov, nil
}

func (e *Evaluator) evaluateDirection(ctx context.Context, branch config.BranchConfig, lec legEvalContext) (model.BranchResult, []model.RouteCoverageRow, error) {
	res := model.BranchResult{BranchID: branch.ID, BranchName: branch.Name, Status: model.StatusUnavailable}
	var coverageRows []model.RouteCoverageRow
	prefMode := e.Config.Constraints.AirlinePreferenceMode

	for _, leg := range branch.Legs {
		if e.Verbose && provider.WarnSUPreference(leg.PreferredAirlines, e.Config.Providers, e.Registry) {
			log.Printf("warn: preferred SU on %s-%s but no provider declares SU; continuing with advisory selection", leg.From, leg.To)
		}
	}

	res.VisaCategory = model.VisaCategory(scoring.WorstVisaFromLegs(branch.Legs))
	disruption := scoring.WorstOperationalDisruptionFromLegs(branch.Legs, e.Config.Risk.OperationalDisruption)
	res.OperationalDisruptionRisk = disruption
	res.OperationalDisruptionNotes = model.OperationalDisruptionNotes(disruption)

	legOffers := make([][]model.Offer, len(branch.Legs))
	providersUsed := map[string]bool{}
	window := e.Config.Coverage.WindowDays

	for i, leg := range branch.Legs {
		legStart := time.Now()
		e.progressLegStart(lec.direction, i+1, len(branch.Legs), leg.From, leg.To)
		p, ok := provider.SelectProvider(leg, e.Config.Providers, e.Registry, prefMode)
		if e.MockOnly {
			p, ok = e.Registry.Get("mock")
		}
		if !ok || p == nil {
			e.progressLegDone(0, time.Since(legStart))
			res.ReasonCodes = []model.ReasonCode{model.ReasonNoProvider}
			return res, coverageRows, nil
		}

		targetDate := lec.tripDate
		notBefore := ""
		var prevOffers []model.Offer
		if i > 0 {
			prevOffers = legOffers[i-1]
			targetDate = legTargetDate(prevOffers, lec.tripDate)
			notBefore = earliestLeg2CalendarDate(prevOffers, branch.MinConnectionHours)
		}
		covQ := baseQuery(e, leg, targetDate, window)
		covQ.NotBeforeDate = notBefore
		if ca, ok := p.(provider.CoverageAnalyzer); ok {
			row, err := ca.AnalyzeLegCoverage(ctx, covQ)
			if err != nil {
				e.progressLegDone(0, time.Since(legStart))
				res.ReasonCodes = []model.ReasonCode{model.ReasonAPIError}
				return res, coverageRows, nil
			}
			row.BranchID = branch.ID
			row.BranchName = branch.Name
			if e.Config.RoundTrip() {
				row.Direction = lec.direction
			}
			row.LegIndex = i
			coverageRows = append(coverageRows, row)
		}

		searchDates := legSearchDates(i, lec.tripDate, prevOffers, branch, lec.firstLegDates)
		var allOffers []model.Offer
		for _, d := range searchDates {
			q := baseQuery(e, leg, d, window)
			q.TargetDate = targetDate
			q.NotBeforeDate = notBefore
			offers, err := p.Search(ctx, q)
			if err != nil {
				e.progressLegDone(0, time.Since(legStart))
				res.ReasonCodes = []model.ReasonCode{model.ReasonAPIError}
				return res, coverageRows, nil
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
				e.progressLegDone(0, time.Since(legStart))
				res.ReasonCodes = []model.ReasonCode{model.ReasonAPIError}
				return res, coverageRows, nil
			}
			allOffers = append(allOffers, offers...)
			if len(offers) > 0 && len(coverageRows) > 0 {
				idx := len(coverageRows) - 1
				coverageRows[idx].SelectedDate = offers[0].SearchDate
				coverageRows[idx].EstimatedDate = offers[0].EstimatedDate
			}
		}

		if i > 0 {
			allOffers = filterLeg2AfterEarliest(allOffers, prevOffers, branch.MinConnectionHours)
		}

		if len(allOffers) == 0 {
			if row, ok := coverageRowForLeg(coverageRows, i, lec.direction); ok {
				cur := e.Config.Scoring.Currency
				if cur == "" {
					cur = "USD"
				}
				allOffers = offersFromCoverage(row, leg.From, leg.To, targetDate, cur)
				if len(allOffers) > 0 && len(coverageRows) > 0 {
					idx := len(coverageRows) - 1
					if allOffers[0].SearchDate != "" {
						coverageRows[idx].SelectedDate = allOffers[0].SearchDate
					}
					coverageRows[idx].EstimatedDate = allOffers[0].EstimatedDate
				}
			}
		}

		if len(allOffers) == 0 {
			e.progressLegDone(0, time.Since(legStart))
			return e.finishPartialBranch(ctx, res, branch, coverageRows, legOffers, providersUsed, disruption, lec.tripDate, lec.direction)
		}
		allOffers = preferTargetDateOffers(allOffers, targetDate)
		legOffers[i] = allOffers
		legAirlines := collectLegAirlines(allOffers, targetDate)
		res.LegAirlines = append(res.LegAirlines, model.LegAirlines{
			From:              leg.From,
			To:                leg.To,
			TargetDate:        targetDate,
			AvailableAirlines: legAirlines,
		})
		for j := range coverageRows {
			if coverageRows[j].LegIndex == i {
				coverageRows[j].AvailableAirlines = model.MergeAirlineLists(coverageRows[j].AvailableAirlines, legAirlines)
				break
			}
		}
		providersUsed[p.Name()] = true
		e.logLegOffers(leg, allOffers, p.Name(), searchDates)
		e.progressLegDone(len(allOffers), time.Since(legStart))
	}

	combined, ok := combineLegOffers(branch, legOffers, e.Config.Scoring.Currency)
	if !ok {
		return e.finishPartialBranch(ctx, res, branch, coverageRows, legOffers, providersUsed, disruption, lec.tripDate, lec.direction)
	}

	combined.BranchID = branch.ID
	combined.SelfTransfer = branch.Type == "self_transfer"
	applyVisaToOffer(&combined, branch)

	if combined.ConnectionVerified {
		if codes := checkOfferConnections(combined, branch); len(codes) > 0 {
			res.Status = model.StatusRejected
			res.ReasonCodes = codes
			return res, coverageRows, nil
		}
	} else {
		combined.Notes = appendNote(combined.Notes, model.NoteConnectionUnverified)
	}

	if reject, codes := scoring.CheckBaggage(combined.CheckedBaggageKg, combined.DataQuality, e.Config.Constraints); reject {
		res.Status = model.StatusRejected
		res.ReasonCodes = codes
		return res, coverageRows, nil
	}

	if err := scoring.NormalizeOfferPrice(&combined, e.Config.Scoring.Currency); err != nil {
		res.ReasonCodes = []model.ReasonCode{model.ReasonCurrencyUnconvertible}
		return res, coverageRows, nil
	}

	if combined.PriceNormalized == nil {
		res.ReasonCodes = []model.ReasonCode{model.ReasonCurrencyUnconvertible}
		return res, coverageRows, nil
	}

	score, bd := scoring.ScoreOffer(combined, branch, e.Config.Scoring, e.Config.Constraints, scoring.WorstVisaFromLegs(branch.Legs), disruption, e.Config.Risk.OperationalDisruption)
	if branchUsesPartialData(combined) {
		res.Status = model.StatusPartial
		res.ReasonCodes = appendReasonCode(res.ReasonCodes, model.ReasonPartialData)
	} else {
		res.Status = model.StatusOK
	}
	res.Offer = &combined
	res.Score = &score
	res.Breakdown = &bd
	res.OperationalDisruptionPenalty = bd.RegionalDisruptionPenalty
	res.PriceComparison = e.computePriceComparison(ctx, branch, &combined)
	for p := range providersUsed {
		res.Providers = append(res.Providers, p)
	}
	sort.Strings(res.Providers)
	return res, coverageRows, nil
}

func preferTargetDateOffers(offers []model.Offer, targetDate string) []model.Offer {
	var exact []model.Offer
	for _, o := range offers {
		if o.SearchDate == targetDate && !o.EstimatedDate {
			exact = append(exact, o)
		}
	}
	if len(exact) > 0 {
		return exact
	}
	return offers
}

func tagOffersForLeg(offers []model.Offer, searchDate, targetDate string) []model.Offer {
	out := make([]model.Offer, len(offers))
	for i, o := range offers {
		o.SearchDate = searchDate
		if searchDate != targetDate && !o.EstimatedDate {
			o.EstimatedDate = true
			o.Notes = appendNote(o.Notes, model.NoteEstimatedDate)
		}
		out[i] = o
	}
	return out
}

func baseQuery(e *Evaluator, leg config.LegConfig, date string, window int) model.Query {
	return model.Query{
		From:               leg.From,
		To:                 leg.To,
		Date:               date,
		TargetDate:         date,
		Passengers:         e.Config.Trip.Passengers,
		Cabin:              e.Config.Trip.Cabin,
		Airlines:           leg.PreferredAirlines,
		CoverageWindowDays: window,
	}
}

func legTargetDate(prevLegOffers []model.Offer, tripDate string) string {
	return leg2TargetDate(prevLegOffers, tripDate)
}

func checkOfferConnections(offer model.Offer, branch config.BranchConfig) []model.ReasonCode {
	if len(offer.Segments) < 2 {
		connH := offer.ConnectionDuration.Hours()
		return scoring.CheckConnection(connH, branch)
	}
	for i := 0; i < len(offer.Segments)-1; i++ {
		connH := scoring.ConnectionDuration(offer.Segments[i].Arrival, offer.Segments[i+1].Departure).Hours()
		if codes := scoring.CheckConnection(connH, branch); len(codes) > 0 {
			return codes
		}
	}
	return nil
}

func combineLegOffers(branch config.BranchConfig, legs [][]model.Offer, scoringCurrency string) (model.Offer, bool) {
	if len(legs) == 0 || len(legs[0]) == 0 {
		return model.Offer{}, false
	}
	if len(legs) == 1 {
		o := legs[0][0]
		o.LegDetails = []model.LegDetail{legDetailFromOffer(o, branch.Legs[0].From, branch.Legs[0].To)}
		o.ConnectionVerified = o.TimingVerified
		return o, true
	}

	best := model.Offer{}
	found := false
	var bestPrice int64

	var search func(legIdx int, current model.Offer, hasCurrent bool)
	search = func(legIdx int, current model.Offer, hasCurrent bool) {
		if legIdx == len(legs) {
			if !hasCurrent {
				return
			}
			var sum int64
			for _, ld := range current.LegDetails {
				n, err := scoring.NormalizeMoney(ld.Price, scoringCurrency)
				if err != nil {
					return
				}
				sum += n.Amount
			}
			if sum == 0 {
				n, err := scoring.NormalizeMoney(current.Price, scoringCurrency)
				if err != nil {
					return
				}
				sum = n.Amount
			}
			norm := model.Money{Amount: sum, Currency: scoringCurrency}
			candidate := current
			candidate.PriceNormalized = &norm
			if !found || sum < bestPrice {
				best = candidate
				bestPrice = sum
				found = true
			}
			return
		}
		for _, o := range legs[legIdx] {
			var candidate model.Offer
			var ok bool
			if legIdx == 0 {
				candidate = o
				candidate.LegDetails = []model.LegDetail{legDetailFromOffer(o, branch.Legs[0].From, branch.Legs[0].To)}
				candidate.ConnectionVerified = o.TimingVerified
				ok = true
			} else {
				candidate, ok = attachLegOffer(current, o, branch, legIdx)
			}
			if !ok {
				continue
			}
			search(legIdx+1, candidate, true)
		}
	}
	search(0, model.Offer{}, false)
	return best, found
}

func attachLegOffer(prev, next model.Offer, branch config.BranchConfig, legIdx int) (model.Offer, bool) {
	if len(prev.Segments) == 0 || len(next.Segments) == 0 {
		return model.Offer{}, false
	}
	lastPrev := prev.Segments[len(prev.Segments)-1]
	firstNext := next.Segments[0]
	if firstNext.Departure.Before(lastPrev.Arrival) {
		return model.Offer{}, false
	}
	verified := prev.TimingVerified && next.TimingVerified && !prev.EstimatedDate && !next.EstimatedDate
	conn := scoring.ConnectionDuration(lastPrev.Arrival, firstNext.Departure)
	if verified {
		connH := conn.Hours()
		if connH < branch.MinConnectionHours || connH > branch.MaxConnectionHours {
			return model.Offer{}, false
		}
	} else if conn <= 0 {
		conn = time.Duration(branch.MinConnectionHours) * time.Hour
	}

	legCfg := branch.Legs[legIdx]
	details := append(append([]model.LegDetail{}, prev.LegDetails...), legDetailFromOffer(next, legCfg.From, legCfg.To))
	price := model.Money{Amount: prev.Price.Amount + next.Price.Amount, Currency: prev.Price.Currency}
	if prev.Price.Currency != next.Price.Currency {
		price.Currency = prev.Price.Currency
	}
	notes := mergeNotes(prev.Notes, next.Notes)
	if !verified {
		notes = appendNote(notes, model.NoteConnectionUnverified)
	}
	connectionVerified := prev.ConnectionVerified && verified

	return model.Offer{
		Provider:           fmt.Sprintf("%s+%s", prev.Provider, next.Provider),
		Segments:           append(append([]model.Segment{}, prev.Segments...), next.Segments...),
		Price:              price,
		TotalDuration:      prev.TotalDuration + conn + next.TotalDuration,
		ConnectionDuration: conn,
		CheckedBaggageKg:   minBaggage(prev.CheckedBaggageKg, next.CheckedBaggageKg),
		VisaRisk:           maxRisk(prev.VisaRisk, next.VisaRisk),
		DataQuality:        mergeDataQuality(prev.DataQuality, next.DataQuality),
		Notes:              notes,
		TimingVerified:     prev.TimingVerified && next.TimingVerified,
		ConnectionVerified: connectionVerified,
		EstimatedDate:      prev.EstimatedDate || next.EstimatedDate,
		LegDetails:         details,
	}, true
}

func mergeOffers(o1, o2 model.Offer, conn time.Duration, branch config.BranchConfig, connectionVerified bool) model.Offer {
	segs := append([]model.Segment{}, o1.Segments...)
	segs = append(segs, o2.Segments...)
	price := model.Money{
		Amount:   o1.Price.Amount + o2.Price.Amount,
		Currency: o1.Price.Currency,
	}
	if o1.Price.Currency != o2.Price.Currency {
		price.Currency = o1.Price.Currency
	}
	bag := minBaggage(o1.CheckedBaggageKg, o2.CheckedBaggageKg)
	dq := mergeDataQuality(o1.DataQuality, o2.DataQuality)
	notes := mergeNotes(o1.Notes, o2.Notes)
	if !connectionVerified {
		notes = appendNote(notes, model.NoteConnectionUnverified)
	}
	return model.Offer{
		Provider:           fmt.Sprintf("%s+%s", o1.Provider, o2.Provider),
		Segments:           segs,
		Price:              price,
		TotalDuration:      o1.TotalDuration + conn + o2.TotalDuration,
		ConnectionDuration: conn,
		CheckedBaggageKg:   bag,
		VisaRisk:           maxRisk(o1.VisaRisk, o2.VisaRisk),
		DataQuality:        dq,
		Notes:              notes,
		TimingVerified:     o1.TimingVerified && o2.TimingVerified,
		ConnectionVerified: connectionVerified,
		EstimatedDate:      o1.EstimatedDate || o2.EstimatedDate,
		LegDetails: []model.LegDetail{
			legDetailFromOffer(o1, branch.Legs[0].From, branch.Legs[0].To),
			legDetailFromOffer(o2, branch.Legs[1].From, branch.Legs[1].To),
		},
	}
}

func appendNote(notes []string, note string) []string {
	for _, n := range notes {
		if n == note {
			return notes
		}
	}
	return append(notes, note)
}

func (e *Evaluator) logLegOffers(leg config.LegConfig, offers []model.Offer, providerName string, searchDates []string) {
	if !e.Verbose {
		return
	}
	fmt.Printf("\n[leg] %s→%s search_dates=%v provider=%s offers=%d\n",
		leg.From, leg.To, searchDates, providerName, len(offers))
	for _, o := range offers {
		airline, fn := "", ""
		segs := len(o.Segments)
		if segs > 0 {
			airline = o.Segments[0].Airline
			fn = o.Segments[0].FlightNumber
		}
		fmt.Printf("[%s]\n", providerName)
		fmt.Printf("%s -> %s\n", leg.From, leg.To)
		if o.SearchDate != "" {
			fmt.Printf("search_date: %s\n", o.SearchDate)
		}
		fmt.Printf("price: %s\n", formatLegPrice(o.Price))
		fmt.Printf("airline: %s\n", airline)
		fmt.Printf("flight: %s\n", formatFlightLabel(airline, fn))
		fmt.Printf("segments: %d\n", segs)
		fmt.Printf("timing_verified: %v\n", o.TimingVerified)
	}
}

func mergeDataQuality(a, b model.DataQuality) model.DataQuality {
	if a == model.DataQualityBrowserCollected || b == model.DataQualityBrowserCollected {
		return model.DataQualityBrowserCollected
	}
	if a == model.DataQualityCached || b == model.DataQualityCached {
		return model.DataQualityCached
	}
	if a == model.DataQualityMock || b == model.DataQualityMock {
		return model.DataQualityMock
	}
	if a != "" {
		return a
	}
	return b
}

func mergeNotes(a, b []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(a)+len(b))
	for _, n := range append(a, b...) {
		if n == "" || seen[n] {
			continue
		}
		seen[n] = true
		out = append(out, n)
	}
	return out
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

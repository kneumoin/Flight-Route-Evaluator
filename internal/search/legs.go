package search

import (
	"fmt"
	"time"

	"github.com/kneumoin/nepal/internal/config"
	"github.com/kneumoin/nepal/internal/model"
	"github.com/kneumoin/nepal/internal/scoring"
)

func addDays(dateStr string, days int) string {
	t, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		return dateStr
	}
	return t.AddDate(0, 0, days).Format("2006-01-02")
}

func legSearchDates(legIndex int, tripDate string, leg1Offers []model.Offer, branch config.BranchConfig, firstLegDates []string) []string {
	if legIndex == 0 && len(firstLegDates) > 0 {
		return append([]string(nil), firstLegDates...)
	}
	if legIndex == 0 {
		return []string{tripDate}
	}
	return leg2SearchDates(tripDate, leg1Offers, branch)
}

func leg2SearchDates(tripDate string, leg1Offers []model.Offer, branch config.BranchConfig) []string {
	anchor := tripDate
	if len(leg1Offers) > 0 {
		if d := hubArrivalDate(leg1Offers[0]); d != "" {
			anchor = d
		}
	}

	if len(leg1Offers) == 0 {
		return uniqueDates([]string{anchor, addDays(anchor, 1), addDays(anchor, 2)})
	}

	verified := leg1TimingVerified(leg1Offers)
	if !verified {
		return uniqueDates([]string{anchor, addDays(anchor, 1), addDays(anchor, 2)})
	}

	seen := map[string]bool{}
	var dates []string
	add := func(d string) {
		if d == "" || seen[d] {
			return
		}
		seen[d] = true
		dates = append(dates, d)
	}

	for _, o := range leg1Offers {
		if len(o.Segments) == 0 {
			continue
		}
		last := o.Segments[len(o.Segments)-1]
		arrLocal := last.Arrival
		if loc, err := scoring.AirportLocation(last.To); err == nil {
			arrLocal = last.Arrival.In(loc)
		}
		d := arrLocal.Format("2006-01-02")
		add(d)
		add(addDays(d, 1))
		if arrLocal.Hour() >= 20 {
			add(addDays(d, 2))
		}
	}
	if len(dates) == 0 {
		return uniqueDates([]string{anchor, addDays(anchor, 1)})
	}
	return dates
}

func hubArrivalDate(leg1 model.Offer) string {
	if len(leg1.Segments) == 0 {
		return ""
	}
	last := leg1.Segments[len(leg1.Segments)-1]
	if last.Arrival.IsZero() {
		return ""
	}
	loc, err := scoring.AirportLocation(last.To)
	if err != nil {
		return last.Arrival.Format("2006-01-02")
	}
	return last.Arrival.In(loc).Format("2006-01-02")
}

func earliestLeg2CalendarDate(leg1Offers []model.Offer, minConnHours float64) string {
	earliest := earliestLeg2Departure(leg1Offers, minConnHours)
	if !earliest.IsZero() {
		return earliest.Format("2006-01-02")
	}
	if len(leg1Offers) > 0 {
		return hubArrivalDate(leg1Offers[0])
	}
	return ""
}

func leg2TargetDate(leg1Offers []model.Offer, tripDate string) string {
	if d := hubArrivalDateFromOffers(leg1Offers); d != "" {
		return d
	}
	return tripDate
}

func hubArrivalDateFromOffers(offers []model.Offer) string {
	for _, o := range offers {
		if d := hubArrivalDate(o); d != "" {
			return d
		}
	}
	return ""
}

func uniqueDates(in []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(in))
	for _, d := range in {
		if d == "" || seen[d] {
			continue
		}
		seen[d] = true
		out = append(out, d)
	}
	return out
}

func tagOffersSearchDate(offers []model.Offer, searchDate string) []model.Offer {
	out := make([]model.Offer, len(offers))
	for i, o := range offers {
		o.SearchDate = searchDate
		out[i] = o
	}
	return out
}

func filterLeg2AfterEarliest(offers []model.Offer, leg1Offers []model.Offer, minConnHours float64) []model.Offer {
	notBefore := earliestLeg2CalendarDate(leg1Offers, minConnHours)
	if notBefore == "" {
		return offers
	}
	earliest := earliestLeg2Departure(leg1Offers, minConnHours)
	out := make([]model.Offer, 0, len(offers))
	for _, o := range offers {
		if leg2DepartureAllowed(o, notBefore, earliest) {
			out = append(out, o)
		}
	}
	return out
}

func leg2DepartureAllowed(o model.Offer, notBeforeDate string, earliest time.Time) bool {
	depDate := o.SearchDate
	var depTime time.Time
	if len(o.Segments) > 0 && !o.Segments[0].Departure.IsZero() {
		depTime = o.Segments[0].Departure
		depDate = depTime.Format("2006-01-02")
	}
	if depDate != "" && depDate < notBeforeDate {
		return false
	}
	if !earliest.IsZero() && !depTime.IsZero() && depTime.Before(earliest) {
		return false
	}
	return true
}

func earliestLeg2Departure(leg1Offers []model.Offer, minConnHours float64) time.Time {
	var earliest time.Time
	minConn := time.Duration(minConnHours * float64(time.Hour))
	for _, o := range leg1Offers {
		if !o.TimingVerified || len(o.Segments) == 0 {
			continue
		}
		arr := o.Segments[len(o.Segments)-1].Arrival
		candidate := arr.Add(minConn)
		if earliest.IsZero() || candidate.Before(earliest) {
			earliest = candidate
		}
	}
	return earliest
}

func leg1TimingVerified(offers []model.Offer) bool {
	for _, o := range offers {
		if o.TimingVerified {
			return true
		}
	}
	return false
}

func legDetailFromOffer(o model.Offer, from, to string) model.LegDetail {
	d := model.LegDetail{
		From:           from,
		To:             to,
		SearchDate:     o.SearchDate,
		Price:          o.Price,
		Provider:       o.Provider,
		TimingVerified: o.TimingVerified,
		EstimatedDate:  o.EstimatedDate,
		Transfers:      o.Transfers,
	}
	if len(o.Segments) > 0 {
		s := o.Segments[0]
		last := o.Segments[len(o.Segments)-1]
		d.Airline = s.Airline
		d.FlightNumber = s.FlightNumber
		d.Departure = s.Departure
		d.Arrival = last.Arrival
		if d.SearchDate == "" && !s.Departure.IsZero() {
			d.SearchDate = s.Departure.Format("2006-01-02")
		}
	}
	d.AvailableAirlines = append([]string(nil), o.AvailableAirlines...)
	if len(d.AvailableAirlines) == 0 && d.Airline != "" {
		d.AvailableAirlines = []string{d.Airline}
	}
	return d
}

func formatLegPrice(m model.Money) string {
	return fmt.Sprintf("%.2f %s", float64(m.Amount)/100, m.Currency)
}

func formatFlightLabel(airline, flightNumber string) string {
	if flightNumber != "" {
		return flightNumber
	}
	return airline
}

// CollectLegAirlinesForTest exposes leg airline aggregation for tests.
func CollectLegAirlinesForTest(offers []model.Offer, targetDate string) []string {
	return collectLegAirlines(offers, targetDate)
}

// collectLegAirlines gathers unique carriers from API offers on the leg target date.
func collectLegAirlines(offers []model.Offer, targetDate string) []string {
	var lists [][]string
	for _, o := range offers {
		if o.SearchDate != "" && o.SearchDate != targetDate && !o.EstimatedDate {
			continue
		}
		if len(o.AvailableAirlines) > 0 {
			lists = append(lists, o.AvailableAirlines)
			continue
		}
		if len(o.Segments) > 0 {
			if a := o.Segments[0].Airline; a != "" {
				lists = append(lists, []string{a})
			}
		}
	}
	return model.MergeAirlineLists(lists...)
}

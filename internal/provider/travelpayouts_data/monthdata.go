package travelpayouts_data

import (
	"encoding/json"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/kneumoin/nepal/internal/model"
)

const (
	endpointCalendar    = "v1/prices/calendar"
	endpointMonthMatrix = "v2/prices/month-matrix"
)

type dayPrice struct {
	Date         string
	Price        float64
	Airline      string
	FlightNumber string
	Transfers    *int
	DepartureAt  string
	Source       string
}

type monthPriceIndex struct {
	ByDate map[string]dayPrice
}

func parseCalendarResponse(raw []byte) (monthPriceIndex, error) {
	var payload struct {
		Success bool                       `json:"success"`
		Data    map[string]calendarEntry `json:"data"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return monthPriceIndex{}, err
	}
	idx := monthPriceIndex{ByDate: make(map[string]dayPrice, len(payload.Data))}
	for date, e := range payload.Data {
		if e.Price <= 0 {
			continue
		}
		dp := dayPrice{
			Date:         date,
			Price:        e.Price,
			Airline:      strings.ToUpper(strings.TrimSpace(e.Airline)),
			FlightNumber: formatFlightNumber(e.FlightNumber),
			DepartureAt:  e.DepartureAt,
			Source:       endpointCalendar,
		}
		if e.Transfers >= 0 {
			t := e.Transfers
			dp.Transfers = &t
		}
		idx.ByDate[date] = dp
	}
	return idx, nil
}

type calendarEntry struct {
	Price        float64     `json:"price"`
	Airline      string      `json:"airline"`
	FlightNumber json.Number `json:"flight_number"`
	DepartureAt  string      `json:"departure_at"`
	Transfers    int         `json:"transfers"`
}

func parseMonthMatrixResponse(raw []byte) (monthPriceIndex, error) {
	var payload struct {
		Success bool               `json:"success"`
		Data    []monthMatrixEntry `json:"data"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return monthPriceIndex{}, err
	}
	idx := monthPriceIndex{ByDate: make(map[string]dayPrice)}
	for _, e := range payload.Data {
		if e.DepartDate == "" || e.Value <= 0 {
			continue
		}
		existing, ok := idx.ByDate[e.DepartDate]
		if ok && existing.Price <= e.Value {
			continue
		}
		dp := dayPrice{
			Date:   e.DepartDate,
			Price:  e.Value,
			Source: endpointMonthMatrix,
		}
		if e.NumberOfChanges >= 0 {
			t := e.NumberOfChanges
			dp.Transfers = &t
		}
		idx.ByDate[e.DepartDate] = dp
	}
	return idx, nil
}

type monthMatrixEntry struct {
	DepartDate      string  `json:"depart_date"`
	Value           float64 `json:"value"`
	NumberOfChanges int     `json:"number_of_changes"`
}

func mergeMonthIndexes(calendar, matrix monthPriceIndex) monthPriceIndex {
	out := monthPriceIndex{ByDate: make(map[string]dayPrice)}
	for d, e := range calendar.ByDate {
		out.ByDate[d] = e
	}
	for d, e := range matrix.ByDate {
		cur, ok := out.ByDate[d]
		if !ok {
			out.ByDate[d] = e
			continue
		}
		if e.Price < cur.Price && cur.Airline == "" {
			out.ByDate[d] = e
		}
	}
	return out
}

func (idx monthPriceIndex) coverageDays() int {
	return len(idx.ByDate)
}

func (idx monthPriceIndex) cheapestInWindow(target string, window int, notBefore string) (dayPrice, bool) {
	targetT, err := time.Parse("2006-01-02", target)
	if err != nil {
		return dayPrice{}, false
	}
	var best dayPrice
	found := false
	for date, e := range idx.ByDate {
		if notBefore != "" && date < notBefore {
			continue
		}
		dist := dayDistance(targetT, date)
		if dist > window {
			continue
		}
		if !found || e.Price < best.Price || (e.Price == best.Price && dist < dayDistance(targetT, best.Date)) {
			best = e
			found = true
		}
	}
	return best, found
}

func (idx monthPriceIndex) nearestInWindow(target string, window int, notBefore string) (dayPrice, bool) {
	targetT, err := time.Parse("2006-01-02", target)
	if err != nil {
		return dayPrice{}, false
	}
	var best dayPrice
	bestDist := window + 1
	found := false
	for date, e := range idx.ByDate {
		if notBefore != "" && date < notBefore {
			continue
		}
		dist := dayDistance(targetT, date)
		if dist > window {
			continue
		}
		if !found || dist < bestDist || (dist == bestDist && e.Price < best.Price) {
			best = e
			bestDist = dist
			found = true
		}
	}
	return best, found
}

func dayDistance(target time.Time, dateStr string) int {
	d, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		return 9999
	}
	dist := int(d.Sub(target).Hours() / 24)
	if dist < 0 {
		return -dist
	}
	return dist
}

func monthKey(date string) string {
	if len(date) >= 7 {
		return date[:7]
	}
	return date
}

func offerFromDayPrice(q model.Query, dp dayPrice, targetDate string, currency string) *model.Offer {
	cur := strings.ToUpper(currency)
	money := model.Money{Amount: int64(math.Round(dp.Price * 100)), Currency: cur}
	segDate := dp.Date
	timingVerified := dp.DepartureAt != ""
	seg := buildSegment(q.From, q.To, segDate, dp.DepartureAt, dp.Airline, dp.FlightNumber, 0)
	totalDur := seg.Duration
	if totalDur == 0 {
		totalDur = defaultLegDuration(q.From, q.To)
		seg = buildSegment(q.From, q.To, segDate, "", dp.Airline, dp.FlightNumber, int(totalDur.Hours()))
	}
	estimated := segDate != targetDate
	notes := append([]string(nil), cachedNotes...)
	if estimated {
		notes = append(notes, model.NoteEstimatedDate)
	}
	return &model.Offer{
		Provider:         "travelpayouts_data",
		Segments:         []model.Segment{seg},
		Price:            money,
		TotalDuration:    totalDur,
		CheckedBaggageKg: nil,
		VisaRisk:         model.RiskLow,
		DataQuality:      model.DataQualityCached,
		Notes:            notes,
		SearchDate:       segDate,
		TimingVerified:   timingVerified,
		EstimatedDate:    estimated,
		Transfers:        dp.Transfers,
	}
}

func coverageRowFromIndex(q model.Query, idx monthPriceIndex, target string, window int, currency string) model.RouteCoverageRow {
	row := model.RouteCoverageRow{
		From:         q.From,
		To:           q.To,
		TargetDate:   target,
		CoverageDays: idx.coverageDays(),
	}
	if cheapest, ok := idx.cheapestInWindow(target, window, q.NotBeforeDate); ok {
		row.CheapestDateNearTarget = cheapest.Date
		m := model.Money{
			Amount:   int64(math.Round(cheapest.Price * 100)),
			Currency: strings.ToUpper(currency),
		}
		row.MinPrice = &m
		row.Airline = cheapest.Airline
		row.Transfers = cheapest.Transfers
	}
	row.AvailableAirlines = model.MergeAirlineLists(idx.airlinesOnDate(target), idx.airlinesInMonth())
	return row
}

func (idx monthPriceIndex) airlinesInMonth() []string {
	var codes []string
	for _, e := range idx.ByDate {
		if a := strings.ToUpper(strings.TrimSpace(e.Airline)); a != "" {
			codes = append(codes, a)
		}
	}
	return model.MergeAirlines(codes...)
}

func sortedDates(idx monthPriceIndex) []string {
	dates := make([]string, 0, len(idx.ByDate))
	for d := range idx.ByDate {
		dates = append(dates, d)
	}
	sort.Strings(dates)
	return dates
}

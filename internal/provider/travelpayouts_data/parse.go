package travelpayouts_data

import (
	"encoding/json"
	"math"
	"strings"
	"time"

	"github.com/kneumoin/nepal/internal/model"
)

var cachedNotes = []string{
	"Cached price data",
	"Baggage unknown",
	"Schedule details may be incomplete",
}

func parseCheapResponse(raw []byte, q model.Query, currency string) (*model.Offer, error) {
	var payload struct {
		Success bool                              `json:"success"`
		Data    map[string]map[string]cheapEntry `json:"data"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, err
	}
	destMap, ok := payload.Data[strings.ToUpper(q.To)]
	if !ok || len(destMap) == 0 {
		return nil, nil
	}

	var best *cheapEntry
	for _, e := range destMap {
		cp := e
		if best == nil || cp.Price < best.Price {
			best = &cp
		}
	}
	if best == nil || best.Price <= 0 {
		return nil, nil
	}

	cur := strings.ToUpper(currency)
	money := model.Money{Amount: int64(math.Round(best.Price * 100)), Currency: cur}
	airline := strings.ToUpper(strings.TrimSpace(best.Airline))
	fn := formatFlightNumber(best.FlightNumber)
	seg := buildSegment(q.From, q.To, q.Date, best.DepartureAt, airline, fn, 0)
	totalDur := seg.Duration
	timingVerified := best.DepartureAt != ""
	if totalDur == 0 {
		totalDur = defaultLegDuration(q.From, q.To)
		seg = buildSegment(q.From, q.To, q.Date, "", airline, fn, int(totalDur.Hours()))
	}

	return &model.Offer{
		Provider:         "travelpayouts_data",
		Segments:         []model.Segment{seg},
		Price:            money,
		TotalDuration:    totalDur,
		CheckedBaggageKg: nil,
		VisaRisk:         model.RiskLow,
		DataQuality:      model.DataQualityCached,
		Notes:            append([]string(nil), cachedNotes...),
		SearchDate:       q.Date,
		TimingVerified:   timingVerified,
	}, nil
}

type cheapEntry struct {
	Price        float64     `json:"price"`
	Airline      string      `json:"airline"`
	FlightNumber json.Number `json:"flight_number"`
	DepartureAt  string      `json:"departure_at"`
}

func parseLatestResponse(raw []byte, q model.Query, currency string) (*model.Offer, error) {
	var payload struct {
		Success bool          `json:"success"`
		Data    []latestEntry `json:"data"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, err
	}

	var best *latestEntry
	for i := range payload.Data {
		e := &payload.Data[i]
		if !strings.EqualFold(e.Origin, q.From) || !strings.EqualFold(e.Destination, q.To) {
			continue
		}
		if e.DepartDate != "" && e.DepartDate != q.Date {
			continue
		}
		if best == nil || e.Value < best.Value {
			best = e
		}
	}
	if best == nil || best.Value <= 0 {
		return nil, nil
	}

	cur := strings.ToUpper(currency)
	money := model.Money{Amount: int64(math.Round(best.Value * 100)), Currency: cur}
	segDate := q.Date
	if best.DepartDate != "" {
		segDate = best.DepartDate
	}
	seg := buildSegment(q.From, q.To, segDate, "", "", "", int(defaultLegDuration(q.From, q.To).Hours()))
	totalDur := seg.Duration
	if totalDur == 0 {
		totalDur = defaultLegDuration(q.From, q.To)
	}

	return &model.Offer{
		Provider:         "travelpayouts_data",
		Segments:         []model.Segment{seg},
		Price:            money,
		TotalDuration:    totalDur,
		CheckedBaggageKg: nil,
		VisaRisk:         model.RiskLow,
		DataQuality:      model.DataQualityCached,
		Notes:            append([]string(nil), cachedNotes...),
		SearchDate:       q.Date,
		TimingVerified:   false,
	}, nil
}

type latestEntry struct {
	Origin       string  `json:"origin"`
	Destination  string  `json:"destination"`
	DepartDate   string  `json:"depart_date"`
	Value        float64 `json:"value"`
	NumberOfChanges int  `json:"number_of_changes"`
}

func formatFlightNumber(n json.Number) string {
	if n == "" {
		return ""
	}
	return n.String()
}

func buildSegment(from, to, date, departureAt, airline, flightNumber string, durHours int) model.Segment {
	dep, arr, dur := segmentTimes(from, to, date, departureAt, durHours)
	return model.Segment{
		From: from, To: to,
		Departure: dep, Arrival: arr,
		Airline: airline, FlightNumber: flightNumber,
		Duration: dur,
	}
}

func segmentTimes(from, to, date, departureAt string, durHours int) (time.Time, time.Time, time.Duration) {
	fromLoc := airportLoc(from)
	toLoc := airportLoc(to)
	day, _ := time.Parse("2006-01-02", date)

	if departureAt != "" {
		if t, err := time.Parse(time.RFC3339, departureAt); err == nil {
			dep := t.In(fromLoc)
			dur := time.Duration(durHours) * time.Hour
			if dur == 0 {
				dur = defaultLegDuration(from, to)
			}
			arr := dep.Add(dur).In(toLoc)
			return dep, arr, dur
		}
	}

	sched := legSchedule(from, to)
	dep := time.Date(day.Year(), day.Month(), day.Day(), sched.depHour, 0, 0, 0, fromLoc)
	dur := time.Duration(sched.durH) * time.Hour
	arr := dep.Add(dur).In(toLoc)
	return dep, arr, dur
}

type legSched struct{ depHour, durH int }

func legSchedule(from, to string) legSched {
	key := from + "-" + to
	if s, ok := legSchedules[key]; ok {
		return s
	}
	return legSched{depHour: 8, durH: 5}
}

var legSchedules = map[string]legSched{
	"MOW-DOH": {8, 5}, "DOH-KTM": {16, 4},
	"MOW-DXB": {9, 5}, "DXB-KTM": {23, 4},
	"MOW-DEL": {7, 6}, "DEL-KTM": {16, 2},
	"MOW-IST": {10, 4}, "IST-KTM": {18, 6},
	"MOW-TFU": {6, 8}, "TFU-KTM": {20, 3},
	"MOW-TAS": {8, 4}, "TAS-DEL": {10, 3},
	"MOW-DYU": {8, 4}, "DYU-DEL": {10, 3},
}

func defaultLegDuration(from, to string) time.Duration {
	return time.Duration(legSchedule(from, to).durH) * time.Hour
}

func airportLoc(iata string) *time.Location {
	// Minimal TZ map for synthetic schedules; scoring package is not imported to avoid cycles.
	tz := map[string]string{
		"MOW": "Europe/Moscow", "DOH": "Asia/Qatar", "DXB": "Asia/Dubai",
		"DEL": "Asia/Kolkata", "IST": "Europe/Istanbul", "TFU": "Asia/Shanghai",
		"KTM": "Asia/Kathmandu",
	}
	if name, ok := tz[iata]; ok {
		if loc, err := time.LoadLocation(name); err == nil {
			return loc
		}
	}
	return time.UTC
}

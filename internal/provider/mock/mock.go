package mock

import (
	"context"
	"fmt"
	"time"

	"github.com/kneumoin/nepal/internal/model"
	"github.com/kneumoin/nepal/internal/provider"
	"github.com/kneumoin/nepal/internal/scoring"
)

type Provider struct{}

func New() *Provider { return &Provider{} }

func (p *Provider) Name() string { return "mock" }

func (p *Provider) Capabilities() provider.ProviderCapabilities {
	return provider.ProviderCapabilities{
		SupportedAirlines: map[string]bool{
			"SU": true, "QR": true, "FZ": true, "AI": true, "6E": true, "TK": true, "CA": true, "3U": true,
		},
		AirlineCoverageMode:     model.CoverageKnown,
		SupportsSelfTransfer:    true,
		SupportsBaggageInfo:     true,
		SupportsRealTimePricing: true,
	}
}

func (p *Provider) Search(ctx context.Context, q model.Query) ([]model.Offer, error) {
	_ = ctx
	key := fmt.Sprintf("%s-%s", q.From, q.To)
	offer, ok := legOffers[key]
	if !ok {
		return nil, nil
	}
	o := offer
	o.Provider = p.Name()
	if len(o.Segments) > 0 && o.Segments[0].Airline != "" {
		o.AvailableAirlines = []string{o.Segments[0].Airline}
	}
	return []model.Offer{o}, nil
}

func (p *Provider) AnalyzeLegCoverage(ctx context.Context, q model.Query) (model.RouteCoverageRow, error) {
	_ = ctx
	key := fmt.Sprintf("%s-%s", q.From, q.To)
	target := q.TargetDate
	if target == "" {
		target = q.Date
	}
	row := model.RouteCoverageRow{From: q.From, To: q.To, TargetDate: target}
	offer, ok := legOffers[key]
	if !ok {
		return row, nil
	}
	row.CoverageDays = 1
	row.CheapestDateNearTarget = target
	price := offer.Price
	row.MinPrice = &price
	if len(offer.Segments) > 0 {
		row.Airline = offer.Segments[0].Airline
		if row.Airline != "" {
			row.AvailableAirlines = []string{row.Airline}
		}
	}
	zero := 0
	row.Transfers = &zero
	return row, nil
}

var legOffers = map[string]model.Offer{}

func InitOffers(date string) {
	legOffers = buildOffers(date)
}

func buildOffers(date string) map[string]model.Offer {
	d, _ := time.Parse("2006-01-02", date)
	day := d
	if d.IsZero() {
		day = time.Date(2026, 9, 28, 0, 0, 0, 0, time.UTC)
	}

	mow, _ := scoring.AirportLocation("MOW")
	doh, _ := scoring.AirportLocation("DOH")
	dxb, _ := scoring.AirportLocation("DXB")
	del, _ := scoring.AirportLocation("DEL")
	ist, _ := scoring.AirportLocation("IST")
	tfu, _ := scoring.AirportLocation("TFU")
	ktm, _ := scoring.AirportLocation("KTM")

	mk := func(from, to string, depH, durH int, fromLoc, toLoc *time.Location, airline, fn string, price int64, cur string, bag int) model.Offer {
		dep := time.Date(day.Year(), day.Month(), day.Day(), depH, 0, 0, 0, fromLoc)
		arr := dep.Add(time.Duration(durH) * time.Hour)
		arr = time.Date(arr.Year(), arr.Month(), arr.Day(), arr.Hour(), arr.Minute(), 0, 0, toLoc)
		b := bag
		return model.Offer{
			Segments: []model.Segment{{
				From: from, To: to, Departure: dep, Arrival: arr,
				Airline: airline, FlightNumber: fn, Duration: time.Duration(durH) * time.Hour,
			}},
			Price:            model.Money{Amount: price, Currency: cur},
			TotalDuration:    time.Duration(durH) * time.Hour,
			CheckedBaggageKg: &b,
			VisaRisk:         model.RiskLow,
			DataQuality:      model.DataQualityMock,
			TimingVerified:   true,
		}
	}

	return map[string]model.Offer{
		"MOW-DOH": mk("MOW", "DOH", 8, 5, mow, doh, "QR", "QR001", 85000, "USD", 30),
		"DOH-KTM": mk("DOH", "KTM", 16, 4, doh, ktm, "QR", "QR002", 0, "USD", 30),
		"MOW-DXB": mk("MOW", "DXB", 9, 5, mow, dxb, "SU", "SU100", 7200000, "RUB", 30),
		"DXB-KTM": mk("DXB", "KTM", 23, 4, dxb, ktm, "FZ", "FZ200", 35000, "USD", 25),
		"MOW-DEL": mk("MOW", "DEL", 7, 6, mow, del, "SU", "SU300", 6800000, "RUB", 30),
		"DEL-KTM": mk("DEL", "KTM", 16, 2, del, ktm, "AI", "AI400", 18000, "USD", 30),
		"MOW-IST": mk("MOW", "IST", 10, 4, mow, ist, "TK", "TK500", 78000, "USD", 30),
		"IST-KTM": mk("IST", "KTM", 18, 6, ist, ktm, "TK", "TK501", 42000, "USD", 30),
		"MOW-TFU": mk("MOW", "TFU", 6, 8, mow, tfu, "CA", "CA600", 65000, "USD", 30),
		"TFU-KTM": mk("TFU", "KTM", 20, 3, tfu, ktm, "CA", "CA601", 28000, "USD", 30),
	}
}

func init() {
	InitOffers("2026-09-28")
}

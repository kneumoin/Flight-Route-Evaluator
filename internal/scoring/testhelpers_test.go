package scoring

import (
	"time"

	"github.com/kneumoin/nepal/internal/config"
	"github.com/kneumoin/nepal/internal/model"
)

func testScoringConfig() config.ScoringConfig {
	return config.ScoringConfig{
		Currency:         "USD",
		LateArrivalAfter: "18:00",
		Weights: config.ScoringWeights{
			Price: 1, Duration: 0.5, Baggage: 0.3, Visa: 1, SelfTransfer: 0.8, LateArrival: 0.4,
		},
	}
}

func testConstraints() config.ConstraintsConfig {
	return config.ConstraintsConfig{
		Baggage: config.BaggageConfig{CheckedRequired: true, MinCheckedKg: 23},
	}
}

func testBranch(typ string) config.BranchConfig {
	return config.BranchConfig{
		ID: "test", Type: typ, MinConnectionHours: 2, MaxConnectionHours: 12,
	}
}

func baseOffer() model.Offer {
	mow, _ := AirportLocation("MOW")
	doh, _ := AirportLocation("DOH")
	ktm, _ := AirportLocation("KTM")
	dep1 := time.Date(2026, 9, 28, 10, 0, 0, 0, mow)
	arr1 := time.Date(2026, 9, 28, 16, 0, 0, 0, doh)
	dep2 := time.Date(2026, 9, 28, 20, 0, 0, 0, doh)
	arr2 := time.Date(2026, 9, 29, 4, 0, 0, 0, ktm)
	bag := 30
	usd := model.Money{Amount: 70000, Currency: "USD"}
	return model.Offer{
		Segments: []model.Segment{
			{From: "MOW", To: "DOH", Departure: dep1, Arrival: arr1, Airline: "QR", FlightNumber: "QR100", Duration: arr1.Sub(dep1)},
			{From: "DOH", To: "KTM", Departure: dep2, Arrival: arr2, Airline: "QR", FlightNumber: "QR200", Duration: arr2.Sub(dep2)},
		},
		Price: usd, PriceNormalized: &usd,
		TotalDuration: 18 * time.Hour, ConnectionDuration: 4 * time.Hour,
		CheckedBaggageKg: &bag, VisaRisk: model.RiskLow,
	}
}

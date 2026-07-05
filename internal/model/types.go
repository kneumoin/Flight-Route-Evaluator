package model

import "time"

type BranchStatus string

const (
	StatusOK          BranchStatus = "ok"
	StatusUnavailable BranchStatus = "unavailable"
	StatusRejected    BranchStatus = "rejected"
)

type Risk string

const (
	RiskLow      Risk = "LOW"
	RiskMedium   Risk = "MEDIUM"
	RiskHigh     Risk = "HIGH"
	RiskRejected Risk = "REJECTED"
)

type ReasonCode string

const (
	ReasonNoProvider            ReasonCode = "NO_PROVIDER"
	ReasonNoOffers              ReasonCode = "NO_OFFERS"
	ReasonConnectionTooShort    ReasonCode = "CONNECTION_TOO_SHORT"
	ReasonConnectionTooLong     ReasonCode = "CONNECTION_TOO_LONG"
	ReasonTransitVisaRequired   ReasonCode = "TRANSIT_VISA_REQUIRED"
	ReasonBaggageUnknown        ReasonCode = "BAGGAGE_UNKNOWN"
	ReasonAPIError              ReasonCode = "API_ERROR"
	ReasonCurrencyUnconvertible ReasonCode = "CURRENCY_UNCONVERTIBLE"
)

type Money struct {
	Amount   int64  `json:"amount"`
	Currency string `json:"currency"`
}

type Segment struct {
	From         string        `json:"from"`
	To           string        `json:"to"`
	Departure    time.Time     `json:"departure"`
	Arrival      time.Time     `json:"arrival"`
	Airline      string        `json:"airline"`
	FlightNumber string        `json:"flight_number"`
	Duration     time.Duration `json:"duration"`
}

type Offer struct {
	BranchID           string        `json:"branch_id"`
	Provider           string        `json:"provider"`
	Segments           []Segment     `json:"segments"`
	Price              Money         `json:"price"`
	PriceNormalized    *Money        `json:"price_normalized,omitempty"`
	TotalDuration      time.Duration `json:"total_duration"`
	ConnectionDuration time.Duration `json:"connection_duration"`
	CheckedBaggageKg   *int          `json:"checked_baggage_kg,omitempty"`
	SelfTransfer       bool          `json:"self_transfer"`
	VisaRisk           Risk          `json:"visa_risk"`
}

type ScoreBreakdown struct {
	Price         float64 `json:"price"`
	Duration      float64 `json:"duration"`
	Baggage       float64 `json:"baggage"`
	Visa          float64 `json:"visa"`
	SelfTransfer  float64 `json:"self_transfer"`
	LateArrival   float64 `json:"late_arrival"`
	Penalties     float64 `json:"penalties"`
	Total         float64 `json:"total"`
}

type BranchResult struct {
	BranchID    string          `json:"branch_id"`
	BranchName  string          `json:"branch_name"`
	Status      BranchStatus    `json:"status"`
	ReasonCodes []ReasonCode    `json:"reason_codes,omitempty"`
	Offer       *Offer          `json:"offer,omitempty"`
	Score       *float64        `json:"score,omitempty"`
	Breakdown   *ScoreBreakdown `json:"breakdown,omitempty"`
	Providers   []string        `json:"providers,omitempty"`
}

type TripMeta struct {
	Origin        string `json:"origin"`
	Destination   string `json:"destination"`
	DepartureDate string `json:"departure_date"`
	Passengers    int    `json:"passengers"`
	Cabin         string `json:"cabin"`
}

type EvaluationResult struct {
	GeneratedAt time.Time      `json:"generated_at"`
	Trip        TripMeta       `json:"trip"`
	Branches    []BranchResult `json:"branches"`
}

type Query struct {
	From       string
	To         string
	Date       string
	Passengers int
	Cabin      string
	Airlines   []string
}

type AirlineCoverageMode string

const (
	CoverageKnown   AirlineCoverageMode = "known"
	CoveragePartial AirlineCoverageMode = "partial"
	CoverageUnknown AirlineCoverageMode = "unknown"
)

package model

import (
	"fmt"
	"time"
)

type BranchStatus string

const (
	StatusOK          BranchStatus = "ok"
	StatusUnavailable BranchStatus = "unavailable"
	StatusRejected    BranchStatus = "rejected"
	StatusPartial     BranchStatus = "partial"
)

type PriceComparison struct {
	PriceTarget        *Money `json:"price_target,omitempty"`
	PriceTargetCurrency string `json:"price_target_currency,omitempty"`
	PriceMinus1        *Money `json:"price_minus_1,omitempty"`
	PricePlus1         *Money `json:"price_plus_1,omitempty"`
	PriceWindowMin     *Money `json:"price_window_min,omitempty"`
	PriceWindowMinDate string `json:"price_window_min_date,omitempty"`
	PriceWindowDays    int    `json:"price_window_days,omitempty"`
}

// FormatPriceComparisonCompact returns e.g. "$1000 (D-1 $1080, D+1 $990, 14d min $950)".
func (pc *PriceComparison) FormatPriceComparisonCompact(locale string) string {
	if pc == nil || pc.PriceTarget == nil {
		return "n/a"
	}
	cur := pc.PriceTarget.Currency
	target := formatUSD(pc.PriceTarget)
	minus1 := formatOpt(pc.PriceMinus1, cur)
	plus1 := formatOpt(pc.PricePlus1, cur)
	winMin := formatOpt(pc.PriceWindowMin, cur)
	days := pc.PriceWindowDays
	if days == 0 {
		days = 14
	}
	if locale == "ru" {
		return fmt.Sprintf("%s (D-1 %s, D+1 %s, мин. %dд %s)", target, minus1, plus1, days, winMin)
	}
	return fmt.Sprintf("%s (D-1 %s, D+1 %s, %dd min %s)", target, minus1, plus1, days, winMin)
}

func formatUSD(m *Money) string {
	if m == nil {
		return "n/a"
	}
	return fmt.Sprintf("$%.0f", float64(m.Amount)/100)
}

func formatOpt(m *Money, cur string) string {
	if m == nil {
		return "n/a"
	}
	if cur == "USD" || m.Currency == "USD" {
		return fmt.Sprintf("$%.0f", float64(m.Amount)/100)
	}
	return fmt.Sprintf("%.0f %s", float64(m.Amount)/100, m.Currency)
}

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
	ReasonPartialData           ReasonCode = "PARTIAL_ROUTE_DATA"
	ReasonNoRouteData           ReasonCode = "NO_ROUTE_DATA"
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

type LegDetail struct {
	From           string    `json:"from"`
	To             string    `json:"to"`
	SearchDate     string    `json:"search_date,omitempty"`
	Airline        string    `json:"airline,omitempty"`
	FlightNumber   string    `json:"flight_number,omitempty"`
	Departure      time.Time `json:"departure,omitempty"`
	Arrival        time.Time `json:"arrival,omitempty"`
	Price          Money     `json:"price"`
	Provider       string    `json:"provider,omitempty"`
	TimingVerified bool      `json:"timing_verified,omitempty"`
	EstimatedDate  bool      `json:"estimated_date,omitempty"`
	Transfers      *int      `json:"transfers,omitempty"`
	AvailableAirlines []string `json:"available_airlines,omitempty"`
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
	DataQuality        DataQuality   `json:"data_quality,omitempty"`
	Notes              []string      `json:"notes,omitempty"`
	SearchDate         string        `json:"search_date,omitempty"`
	TimingVerified     bool          `json:"timing_verified,omitempty"`
	ConnectionVerified bool          `json:"connection_verified,omitempty"`
	EstimatedDate      bool          `json:"estimated_date,omitempty"`
	Transfers          *int          `json:"transfers,omitempty"`
	AvailableAirlines  []string      `json:"available_airlines,omitempty"`
	LegDetails         []LegDetail   `json:"leg_details,omitempty"`
}

const (
	NoteConnectionUnverified = "Connection timing not fully verified"
	NoteEstimatedDate        = "Estimated departure date (cached calendar price)"
	NoteNoPriceOnTarget      = "No price on target date"
	NotePriceFromNearbyDate  = "Price from nearby date (cached calendar)"
	NoteRouteDataIncomplete  = "Incomplete route data; connection not verified"
)

type RouteCoverageRow struct {
	BranchID               string `json:"branch_id"`
	BranchName             string `json:"branch_name"`
	Direction              string `json:"direction,omitempty"`
	LegIndex               int    `json:"leg_index"`
	From                   string `json:"from"`
	To                     string `json:"to"`
	TargetDate             string `json:"target_date"`
	CheapestDateNearTarget string `json:"cheapest_date_near_target,omitempty"`
	MinPrice               *Money `json:"min_price,omitempty"`
	Airline                string `json:"airline,omitempty"`
	Transfers              *int   `json:"transfers,omitempty"`
	CoverageDays           int    `json:"coverage_days"`
	SelectedDate           string `json:"selected_date,omitempty"`
	EstimatedDate          bool   `json:"estimated_date,omitempty"`
	PriceMinus1            *Money `json:"price_minus_1,omitempty"`
	PriceTarget            *Money `json:"price_target,omitempty"`
	PricePlus1             *Money `json:"price_plus_1,omitempty"`
	PriceWindowMin         *Money `json:"price_window_min,omitempty"`
	PriceWindowMinDate     string `json:"price_window_min_date,omitempty"`
	PriceWindowDays        int    `json:"price_window_days,omitempty"`
	Provider               string `json:"provider,omitempty"`
	Notes                  string   `json:"notes,omitempty"`
	AvailableAirlines      []string `json:"available_airlines,omitempty"`
}

type DataQuality string

const (
	DataQualityMock             DataQuality = "mock"
	DataQualityCached           DataQuality = "cached"
	DataQualityRealtime         DataQuality = "realtime"
	DataQualityBrowserCollected DataQuality = "browser_collected"
)

// AllowsUnknownBaggage reports whether unknown baggage may be tolerated when config allows cached/browser providers.
func (dq DataQuality) AllowsUnknownBaggage(allowInConfig bool) bool {
	if !allowInConfig {
		return false
	}
	return dq == DataQualityCached || dq == DataQualityBrowserCollected
}

type ScoreBreakdown struct {
	Price         float64 `json:"price"`
	Duration      float64 `json:"duration"`
	Baggage       float64 `json:"baggage"`
	Visa          float64 `json:"visa"`
	SelfTransfer  float64 `json:"self_transfer"`
	LateArrival   float64 `json:"late_arrival"`
	RegionalDisruptionPenalty float64 `json:"operational_disruption_penalty,omitempty"`
	Penalties               float64 `json:"penalties"`
	Total         float64 `json:"total"`
}

type BranchResult struct {
	BranchID        string           `json:"branch_id"`
	BranchName      string           `json:"branch_name"`
	Status          BranchStatus     `json:"status"`
	ReasonCodes     []ReasonCode     `json:"reason_codes,omitempty"`
	Offer           *Offer           `json:"offer,omitempty"`
	Score           *float64         `json:"score,omitempty"`
	Breakdown       *ScoreBreakdown  `json:"breakdown,omitempty"`
	Providers       []string         `json:"providers,omitempty"`
	VisaCategory    VisaCategory     `json:"visa_category,omitempty"`
	PriceComparison *PriceComparison `json:"price_comparison,omitempty"`
	LegAirlines              []LegAirlines            `json:"leg_airlines,omitempty"`
	OperationalDisruptionRisk    OperationalDisruptionRisk `json:"operational_disruption_risk,omitempty"`
	OperationalDisruptionPenalty float64                   `json:"operational_disruption_penalty,omitempty"`
	OperationalDisruptionNotes   string                    `json:"operational_disruption_notes,omitempty"`
	OutboundOffer                *Offer                    `json:"outbound_offer,omitempty"`
	ReturnOffer                  *Offer                    `json:"return_offer,omitempty"`
}

// LegAirlines lists carriers seen in API data for one branch leg on the target date.
type LegAirlines struct {
	From              string   `json:"from"`
	To                string   `json:"to"`
	TargetDate        string   `json:"target_date"`
	AvailableAirlines []string `json:"available_airlines,omitempty"`
}

type TripMeta struct {
	Origin                string `json:"origin"`
	Destination           string `json:"destination"`
	DepartureDate         string `json:"departure_date"`
	ReturnDate            string `json:"return_date,omitempty"`
	ReturnDateEnd         string `json:"return_date_end,omitempty"`
	OutboundForwardDays   int    `json:"outbound_forward_days,omitempty"`
	Passengers            int    `json:"passengers"`
	Cabin                 string `json:"cabin"`
}

type EvaluationResult struct {
	GeneratedAt   time.Time          `json:"generated_at"`
	Trip          TripMeta           `json:"trip"`
	Branches      []BranchResult     `json:"branches"`
	RouteCoverage []RouteCoverageRow `json:"route_coverage,omitempty"`
}

type Query struct {
	From                  string
	To                    string
	Date                  string
	TargetDate            string
	Passengers            int
	Cabin                 string
	Airlines              []string
	AllowCoverageFallback bool
	CoverageWindowDays    int
	NotBeforeDate         string // earliest allowed departure date (YYYY-MM-DD), e.g. after hub arrival
}

type AirlineCoverageMode string

const (
	CoverageKnown   AirlineCoverageMode = "known"
	CoveragePartial AirlineCoverageMode = "partial"
	CoverageUnknown AirlineCoverageMode = "unknown"
)

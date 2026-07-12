package config

import (
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

var secretInConfigRe = regexp.MustCompile(`(?i)(travelpayouts_token|aviasales_token|api_token|token)\s*:\s*['"]?[a-f0-9]{20,}`)

var iataRe = regexp.MustCompile(`^[A-Z]{3}$`)

type Config struct {
	Trip        TripConfig        `yaml:"trip"`
	Constraints ConstraintsConfig `yaml:"constraints"`
	Coverage    CoverageConfig    `yaml:"coverage"`
	Cache       CacheConfig       `yaml:"cache"`
	History     HistoryConfig     `yaml:"history"`
	Providers   []ProviderConfig  `yaml:"providers"`
	Scoring     ScoringConfig     `yaml:"scoring"`
	Risk        RiskConfig        `yaml:"risk"`
	Branches    []BranchConfig    `yaml:"branches"`
}

type CoverageConfig struct {
	WindowDays          int `yaml:"window_days"`
	OutboundForwardDays int `yaml:"outbound_forward_days"`
}

type TripConfig struct {
	Origin        string `yaml:"origin"`
	Destination   string `yaml:"destination"`
	DepartureDate string `yaml:"departure_date"`
	ReturnDate    string `yaml:"return_date"`
	ReturnDateEnd string `yaml:"return_date_end"`
	Passengers    int    `yaml:"passengers"`
	Cabin         string `yaml:"cabin"`
}

type BaggageConfig struct {
	CheckedRequired              bool `yaml:"checked_required"`
	MinCheckedKg                 int  `yaml:"min_checked_kg"`
	AllowUnknownForCachedProvider bool `yaml:"allow_unknown_for_cached_provider"`
}

type ConstraintsConfig struct {
	MaxStops                int                    `yaml:"max_stops"`
	AvoidTransferVisas      bool                   `yaml:"avoid_transfer_visas"`
	AllowSelfTransfer       bool                   `yaml:"allow_self_transfer"`
	AirlinePreferenceMode   AirlinePreferenceMode  `yaml:"airline_preference_mode"`
	Baggage                 BaggageConfig          `yaml:"baggage"`
}

type AirlinePreferenceMode string

const (
	AirlinePreferenceAdvisory AirlinePreferenceMode = "advisory"
	AirlinePreferenceStrict   AirlinePreferenceMode = "strict"
)

type CacheConfig struct {
	Enabled   bool   `yaml:"enabled"`
	TTL       string `yaml:"ttl"`
	Directory string `yaml:"directory"`
}

type HistoryConfig struct {
	Enabled   bool   `yaml:"enabled"`
	Directory string `yaml:"directory"`
}

type ProviderConfig struct {
	ID      string `yaml:"id"`
	Enabled bool   `yaml:"enabled"`
}

type ScoringWeights struct {
	Price        float64 `yaml:"price"`
	Duration     float64 `yaml:"duration"`
	Baggage      float64 `yaml:"baggage"`
	Visa         float64 `yaml:"visa"`
	SelfTransfer float64 `yaml:"self_transfer"`
	LateArrival  float64 `yaml:"late_arrival"`
}

type ScoringConfig struct {
	Currency         string         `yaml:"currency"`
	Weights          ScoringWeights `yaml:"weights"`
	LateArrivalAfter string         `yaml:"late_arrival_after"`
}

type BranchConfig struct {
	ID                 string     `yaml:"id"`
	Name               string     `yaml:"name"`
	Type               string     `yaml:"type"`
	VisaPolicy         string     `yaml:"visa_policy"`
	MinConnectionHours float64    `yaml:"min_connection_hours"`
	MaxConnectionHours float64    `yaml:"max_connection_hours"`
	Legs               []LegConfig `yaml:"legs"`
}

type LegConfig struct {
	From              string   `yaml:"from"`
	To                string   `yaml:"to"`
	PreferredAirlines []string `yaml:"preferred_airlines"`
	ProviderHint      string   `yaml:"provider_hint"`
}

var validBranchTypes = map[string]bool{
	"single_ticket": true, "mixed_carrier": true, "self_transfer": true,
}

var validVisaPolicies = map[string]bool{
	"airside_only": true, "no_extra_visa": true, "transit_only": true, "transit_or_twov": true,
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if err := scanForSecrets(data); err != nil {
		return nil, err
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse yaml: %w", err)
	}
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func (c *Config) Validate() error {
	if !validIATA(c.Trip.Origin) {
		return fmt.Errorf("invalid trip.origin IATA: %s", c.Trip.Origin)
	}
	if !validIATA(c.Trip.Destination) {
		return fmt.Errorf("invalid trip.destination IATA: %s", c.Trip.Destination)
	}
	if _, err := time.Parse("2006-01-02", c.Trip.DepartureDate); err != nil {
		return fmt.Errorf("invalid departure_date: %w", err)
	}
	if c.Trip.ReturnDate != "" {
		retStart, err := time.Parse("2006-01-02", c.Trip.ReturnDate)
		if err != nil {
			return fmt.Errorf("invalid return_date: %w", err)
		}
		if c.Trip.ReturnDateEnd != "" {
			retEnd, err := time.Parse("2006-01-02", c.Trip.ReturnDateEnd)
			if err != nil {
				return fmt.Errorf("invalid return_date_end: %w", err)
			}
			if retEnd.Before(retStart) {
				return fmt.Errorf("return_date_end must be on or after return_date")
			}
		}
	}
	if c.Coverage.OutboundForwardDays < 0 {
		return fmt.Errorf("coverage.outbound_forward_days must be >= 0")
	}
	if c.Trip.Passengers < 1 {
		return fmt.Errorf("passengers must be >= 1")
	}
	if c.Scoring.Currency == "" {
		c.Scoring.Currency = "USD"
	}
	if c.Constraints.AirlinePreferenceMode == "" {
		c.Constraints.AirlinePreferenceMode = AirlinePreferenceAdvisory
	}
	if c.Constraints.AirlinePreferenceMode != AirlinePreferenceAdvisory &&
		c.Constraints.AirlinePreferenceMode != AirlinePreferenceStrict {
		return fmt.Errorf("invalid airline_preference_mode: %s", c.Constraints.AirlinePreferenceMode)
	}
	if c.Coverage.WindowDays == 0 {
		c.Coverage.WindowDays = 3
	}
	if c.Coverage.WindowDays < 0 {
		return fmt.Errorf("coverage.window_days must be >= 0")
	}
	c.NormalizeRisk()
	if err := validateOperationalDisruption(c.Risk.OperationalDisruption); err != nil {
		return err
	}
	for i, b := range c.Branches {
		if err := validateBranch(b); err != nil {
			return fmt.Errorf("branch[%d] %s: %w", i, b.ID, err)
		}
	}
	if err := validateApprovedHubBranches(c.Branches); err != nil {
		return err
	}
	return nil
}

// RoundTrip reports whether return leg evaluation is enabled.
func (c *Config) RoundTrip() bool {
	return strings.TrimSpace(c.Trip.ReturnDate) != ""
}

// OutboundFirstLegDates returns candidate departure dates for the outbound first leg.
func (c *Config) OutboundFirstLegDates() []string {
	return dateRange(c.Trip.DepartureDate, c.Trip.DepartureDate, c.Coverage.OutboundForwardDays)
}

// ReturnFirstLegDates returns candidate departure dates for the return first leg.
func (c *Config) ReturnFirstLegDates() []string {
	end := c.Trip.ReturnDate
	if c.Trip.ReturnDateEnd != "" {
		end = c.Trip.ReturnDateEnd
	}
	return inclusiveDateRange(c.Trip.ReturnDate, end)
}

func dateRange(start, anchor string, forwardDays int) []string {
	if forwardDays <= 0 {
		return []string{anchor}
	}
	return inclusiveDateRange(start, addDays(start, forwardDays))
}

func inclusiveDateRange(start, end string) []string {
	from, err1 := time.Parse("2006-01-02", start)
	to, err2 := time.Parse("2006-01-02", end)
	if err1 != nil || err2 != nil || to.Before(from) {
		return []string{start}
	}
	var out []string
	for d := from; !d.After(to); d = d.AddDate(0, 0, 1) {
		out = append(out, d.Format("2006-01-02"))
	}
	return out
}

func addDays(date string, days int) string {
	t, err := time.Parse("2006-01-02", date)
	if err != nil {
		return date
	}
	return t.AddDate(0, 0, days).Format("2006-01-02")
}

func validateApprovedHubBranches(branches []BranchConfig) error {
	if len(branches) == 0 {
		return nil
	}
	oneStopSeen := map[string]bool{}
	for _, b := range branches {
		switch len(b.Legs) {
		case 2:
			if err := validateOneStopBranch(b); err != nil {
				return err
			}
			if oneStopSeen[b.Legs[0].To] {
				return fmt.Errorf("duplicate one-stop hub branch: %s", b.Legs[0].To)
			}
			oneStopSeen[b.Legs[0].To] = true
		case 3:
			if err := validateTwoStopBranch(b); err != nil {
				return err
			}
		default:
			return fmt.Errorf("branch %s: expected 2 or 3 legs, got %d", b.ID, len(b.Legs))
		}
	}
	return nil
}

func validateOneStopBranch(b BranchConfig) error {
	if b.Legs[0].From != "MOW" || b.Legs[0].To != b.Legs[1].From || b.Legs[1].To != "KTM" {
		return fmt.Errorf("branch %s: legs must be MOW->HUB and HUB->KTM", b.ID)
	}
	hub := b.Legs[0].To
	if !IsValidHubIATA(hub) {
		return fmt.Errorf("branch %s: hub %s not in approved list", b.ID, hub)
	}
	if b.Legs[0].ProviderHint != "" || b.Legs[1].ProviderHint != "" {
		return fmt.Errorf("branch %s: provider_hint not allowed in default config", b.ID)
	}
	return nil
}

func validateTwoStopBranch(b BranchConfig) error {
	if b.Legs[0].From != "MOW" || b.Legs[2].To != "KTM" {
		return fmt.Errorf("branch %s: legs must be MOW->...->KTM", b.ID)
	}
	if b.Legs[0].To != b.Legs[1].From || b.Legs[1].To != b.Legs[2].From {
		return fmt.Errorf("branch %s: legs must chain MOW->H1->H2->KTM", b.ID)
	}
	first, second := b.Legs[0].To, b.Legs[1].To
	if !IsMultiStopFirstLegHub(first) {
		return fmt.Errorf("branch %s: first stop %s not allowed for two-transfer routes", b.ID, first)
	}
	if second != "DEL" {
		return fmt.Errorf("branch %s: second stop must be DEL, got %s", b.ID, second)
	}
	for _, leg := range b.Legs {
		if leg.ProviderHint != "" {
			return fmt.Errorf("branch %s: provider_hint not allowed in default config", b.ID)
		}
	}
	return nil
}

func validateBranch(b BranchConfig) error {
	if b.ID == "" {
		return fmt.Errorf("missing id")
	}
	if !validBranchTypes[b.Type] {
		return fmt.Errorf("unknown branch type: %s", b.Type)
	}
	if !validVisaPolicies[b.VisaPolicy] {
		return fmt.Errorf("unknown visa_policy: %s", b.VisaPolicy)
	}
	if b.MinConnectionHours < 0 || b.MaxConnectionHours < 0 {
		return fmt.Errorf("negative connection hours")
	}
	if b.MinConnectionHours > b.MaxConnectionHours {
		return fmt.Errorf("min_connection_hours > max_connection_hours")
	}
	if len(b.Legs) == 0 {
		return fmt.Errorf("no legs")
	}
	for _, leg := range b.Legs {
		if !validIATA(leg.From) || !validIATA(leg.To) {
			return fmt.Errorf("invalid leg IATA: %s-%s", leg.From, leg.To)
		}
		if err := validateHubLeg(leg, b.ID); err != nil {
			return err
		}
	}
	return nil
}

func validIATA(code string) bool {
	return iataRe.MatchString(strings.ToUpper(code))
}

func (c *CacheConfig) TTLDuration() (time.Duration, error) {
	if c.TTL == "" {
		return 24 * time.Hour, nil
	}
	return time.ParseDuration(c.TTL)
}

func scanForSecrets(data []byte) error {
	if secretInConfigRe.Match(data) {
		return fmt.Errorf("security: config appears to contain an API token; use TRAVELPAYOUTS_TOKEN environment variable instead")
	}
	return nil
}

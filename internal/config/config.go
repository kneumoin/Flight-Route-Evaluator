package config

import (
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

var iataRe = regexp.MustCompile(`^[A-Z]{3}$`)

type Config struct {
	Trip        TripConfig        `yaml:"trip"`
	Constraints ConstraintsConfig `yaml:"constraints"`
	Cache       CacheConfig       `yaml:"cache"`
	History     HistoryConfig     `yaml:"history"`
	Providers   []ProviderConfig  `yaml:"providers"`
	Scoring     ScoringConfig     `yaml:"scoring"`
	Branches    []BranchConfig    `yaml:"branches"`
}

type TripConfig struct {
	Origin        string `yaml:"origin"`
	Destination   string `yaml:"destination"`
	DepartureDate string `yaml:"departure_date"`
	Passengers    int    `yaml:"passengers"`
	Cabin         string `yaml:"cabin"`
}

type BaggageConfig struct {
	CheckedRequired bool `yaml:"checked_required"`
	MinCheckedKg    int  `yaml:"min_checked_kg"`
}

type ConstraintsConfig struct {
	MaxStops           int           `yaml:"max_stops"`
	AvoidTransferVisas bool          `yaml:"avoid_transfer_visas"`
	AllowSelfTransfer  bool          `yaml:"allow_self_transfer"`
	Baggage            BaggageConfig `yaml:"baggage"`
}

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
	if c.Trip.Passengers < 1 {
		return fmt.Errorf("passengers must be >= 1")
	}
	if c.Scoring.Currency == "" {
		c.Scoring.Currency = "USD"
	}
	for i, b := range c.Branches {
		if err := validateBranch(b); err != nil {
			return fmt.Errorf("branch[%d] %s: %w", i, b.ID, err)
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

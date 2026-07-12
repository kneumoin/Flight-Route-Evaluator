package aviasales_search

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const startURL = "https://tickets-api.travelpayouts.com/search/affiliate/start"

// Client is a demo/production client for Aviasales Flight Search API (2025+).
// Requires approved Search API access (Travelpayouts partner, typically 50k+ MAU).
type Client struct {
	cfg    Config
	client *http.Client
}

func NewClient(cfg Config) *Client {
	if cfg.RealHost == "" {
		cfg.RealHost = "localhost"
	}
	if cfg.UserIP == "" {
		cfg.UserIP = "127.0.0.1"
	}
	if cfg.Locale == "" {
		cfg.Locale = "ru"
	}
	if cfg.MarketCode == "" {
		cfg.MarketCode = "ru"
	}
	if cfg.Currency == "" {
		cfg.Currency = "usd"
	}
	return &Client{
		cfg:    cfg,
		client: &http.Client{Timeout: 90 * time.Second},
	}
}

// StartSearch begins an async flight search. Returns search_id and results_url for polling.
func (c *Client) StartSearch(ctx context.Context, directions []Direction, passengers Passengers, tripClass string) (*StartResponse, error) {
	if c.cfg.Token == "" {
		return nil, fmt.Errorf("TRAVELPAYOUTS_TOKEN not set")
	}
	if c.cfg.Marker == "" {
		return nil, fmt.Errorf("TRAVELPAYOUTS_MARKER not set (partner ID from Travelpayouts dashboard)")
	}
	if tripClass == "" {
		tripClass = "Y"
	}

	reqBody := StartRequest{
		Marker:       c.cfg.Marker,
		Locale:       c.cfg.Locale,
		CurrencyCode: strings.ToUpper(c.cfg.Currency),
		MarketCode:   strings.ToUpper(c.cfg.MarketCode),
		SearchParams: SearchParams{
			Directions: directions,
			TripClass:  tripClass,
			Passengers: passengers,
		},
	}
	reqBody.Signature = BuildSignature(c.cfg.Token, reqBody)

	raw, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, startURL, bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}
	c.setHeaders(req, reqBody.Signature)

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("start search HTTP %d: %s", resp.StatusCode, truncate(string(body), 500))
	}

	var out StartResponse
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, fmt.Errorf("decode start response: %w (body=%s)", err, truncate(string(body), 300))
	}
	if out.SearchID == "" {
		return nil, fmt.Errorf("empty search_id in response: %s", truncate(string(body), 300))
	}
	return &out, nil
}

// PollResults fetches search results until is_over or ctx cancelled.
// First call: lastUpdate=0. Server needs ~30–60s; poll every few seconds.
func (c *Client) PollResults(ctx context.Context, resultsURL, searchID string, lastUpdate int64) (*ResultsResponse, error) {
	base := strings.TrimRight(resultsURL, "/")
	u := base + "/search/affiliate/results"
	payload, _ := json.Marshal(ResultsRequest{
		SearchID:            searchID,
		LastUpdateTimestamp: lastUpdate,
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	c.setHeaders(req, "")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode == http.StatusNotModified {
		return &ResultsResponse{LastUpdateTimestamp: lastUpdate, IsOver: false, Raw: body}, nil
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("poll results HTTP %d: %s", resp.StatusCode, truncate(string(body), 500))
	}

	var parsed struct {
		LastUpdateTimestamp int64 `json:"last_update_timestamp"`
		IsOver              bool  `json:"is_over"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, fmt.Errorf("decode results: %w", err)
	}
	return &ResultsResponse{
		LastUpdateTimestamp: parsed.LastUpdateTimestamp,
		IsOver:              parsed.IsOver,
		Raw:                 body,
	}, nil
}

// BookingURL requests a one-time purchase link for a proposal (call on user click only).
func (c *Client) BookingURL(ctx context.Context, resultsURL, searchID, proposalID string) (*ClickResponse, error) {
	base := strings.TrimRight(resultsURL, "/")
	u := fmt.Sprintf("%s/searches/%s/clicks/%s", base, searchID, proposalID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	c.setHeaders(req, "")
	req.Header.Set("marker", c.cfg.Marker)

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("click HTTP %d: %s", resp.StatusCode, truncate(string(body), 500))
	}
	var out ClickResponse
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, fmt.Errorf("decode click: %w", err)
	}
	return &out, nil
}

func (c *Client) setHeaders(req *http.Request, signature string) {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-affiliate-user-id", c.cfg.Token)
	req.Header.Set("x-real-host", c.cfg.RealHost)
	req.Header.Set("x-user-ip", c.cfg.UserIP)
	if signature != "" {
		req.Header.Set("x-signature", signature)
	}
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

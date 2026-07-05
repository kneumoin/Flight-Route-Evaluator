package aviasales

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/kneumoin/nepal/internal/config"
	"github.com/kneumoin/nepal/internal/model"
	"github.com/kneumoin/nepal/internal/provider"
	"github.com/kneumoin/nepal/internal/cache"
)

type Provider struct {
	token  string
	cache  *cache.Store
	client *http.Client
}

func New(cacheCfg config.CacheConfig) *Provider {
	cs, _ := cache.New(cacheCfg)
	return &Provider{
		token:  os.Getenv("AVIASALES_TOKEN"),
		cache:  cs,
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

func (p *Provider) Name() string { return "aviasales" }

func (p *Provider) Capabilities() provider.ProviderCapabilities {
	return provider.ProviderCapabilities{
		SupportedAirlines: map[string]bool{
			"SU": true, "QR": true, "AI": true, "TK": true,
		},
		AirlineCoverageMode:     model.CoveragePartial,
		SupportsSelfTransfer:    false,
		SupportsBaggageInfo:     false,
		SupportsRealTimePricing: true,
	}
}

func (p *Provider) Search(ctx context.Context, q model.Query) ([]model.Offer, error) {
	if p.token == "" {
		return nil, fmt.Errorf("AVIASALES_TOKEN not set")
	}
	key := fmt.Sprintf("%s-%s-%s-%d-%s", q.From, q.To, q.Date, q.Passengers, q.Cabin)
	fetch := func() ([]byte, error) {
		return p.fetchAPI(ctx, q)
	}
	var raw []byte
	var err error
	if p.cache != nil {
		raw, err = p.cache.Fetch(p.Name(), key, fetch)
	} else {
		raw, err = fetch()
	}
	if err != nil {
		return nil, err
	}
	return parsePrices(raw, q)
}

func (p *Provider) fetchAPI(ctx context.Context, q model.Query) ([]byte, error) {
	u := url.URL{
		Scheme: "https",
		Host:   "api.travelpayouts.com",
		Path:   "/v1/prices/cheap",
	}
	params := url.Values{}
	params.Set("origin", q.From)
	params.Set("destination", q.To)
	params.Set("depart_date", q.Date)
	params.Set("token", p.token)
	u.RawQuery = params.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("aviasales status %d", resp.StatusCode)
	}
	return io.ReadAll(resp.Body)
}

// parsePrices maps Travelpayouts cheap prices response to offers (minimal MVP).
func parsePrices(raw []byte, q model.Query) ([]model.Offer, error) {
	var payload map[string]interface{}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, err
	}
	data, _ := payload["data"].(map[string]interface{})
	if data == nil {
		return nil, nil
	}
	dest, _ := data[q.To].(map[string]interface{})
	if dest == nil {
		return nil, nil
	}
	price, _ := dest["price"].(float64)
	if price == 0 {
		return nil, nil
	}
	offer := model.Offer{
		Provider: "aviasales",
		Price:    model.Money{Amount: int64(price * 100), Currency: "USD"},
		Segments: []model.Segment{{
			From: q.From, To: q.To, Airline: "SU", FlightNumber: "SU000",
		}},
		VisaRisk: model.RiskLow,
	}
	return []model.Offer{offer}, nil
}

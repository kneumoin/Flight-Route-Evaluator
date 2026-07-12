package travelpayouts_data

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/kneumoin/nepal/internal/cache"
	"github.com/kneumoin/nepal/internal/config"
	"github.com/kneumoin/nepal/internal/model"
	"github.com/kneumoin/nepal/internal/provider"
	"github.com/kneumoin/nepal/internal/secrets"
)

const (
	baseHost       = "api.travelpayouts.com"
	endpointCheap  = "v1/prices/cheap"
	endpointLatest = "v2/prices/latest"
)

type Provider struct {
	token           string
	currency        string
	cache           *cache.Store
	client          *http.Client
	verbose         bool
	apiRootOverride string
}

func New(cacheCfg config.CacheConfig, token, currency string, verbose bool) *Provider {
	cs, _ := cache.New(cacheCfg)
	if currency == "" {
		currency = "USD"
	}
	return &Provider{
		token:    token,
		currency: strings.ToLower(currency),
		cache:    cs,
		client:   &http.Client{Timeout: 30 * time.Second},
		verbose:  verbose,
	}
}

func (p *Provider) Name() string { return "travelpayouts_data" }

func (p *Provider) Capabilities() provider.ProviderCapabilities {
	return provider.ProviderCapabilities{
		SupportedAirlines:       nil,
		AirlineCoverageMode:     model.CoverageUnknown,
		SupportsSelfTransfer:    false,
		SupportsBaggageInfo:     false,
		SupportsRealTimePricing: false,
	}
}

func (p *Provider) Search(ctx context.Context, q model.Query) ([]model.Offer, error) {
	if p.token == "" {
		return nil, fmt.Errorf("TRAVELPAYOUTS_TOKEN not set")
	}

	offer, endpoint, rep, err := p.searchCached(ctx, q)
	if p.verbose {
		p.logLegDebug(rep)
	}
	if err != nil {
		return nil, err
	}
	if offer == nil && q.AllowCoverageFallback {
		var fbErr error
		offer, endpoint, fbErr = p.searchCoverageFallback(ctx, q)
		if fbErr != nil {
			return nil, fbErr
		}
		if offer != nil && p.verbose {
			fmt.Printf("travelpayouts_data: coverage fallback %s→%s target=%s selected=%s estimated=%v airline=%s\n",
				q.From, q.To, q.TargetDate, offer.SearchDate, offer.EstimatedDate, offer.Segments[0].Airline)
		}
		if offer != nil {
			p.enrichOfferAirlinesFromIndex(ctx, q, offer)
		}
	}
	if offer == nil {
		return nil, nil
	}
	if p.verbose {
		fmt.Printf("travelpayouts_data: %s %s→%s via %s price=%s %s\n",
			q.Date, q.From, q.To, endpoint, formatMoney(offer.Price), offer.DataQuality)
	}
	return []model.Offer{*offer}, nil
}

func (p *Provider) searchCached(ctx context.Context, q model.Query) (*model.Offer, string, legDebugReport, error) {
	rep := legDebugReport{Query: q}

	endpoints := []struct {
		name      string
		requestURL func(model.Query) string
		fetch     func(context.Context, model.Query) ([]byte, int, error)
		inspect   func([]byte, model.Query) endpointDebug
		parse     func([]byte, model.Query, string) (*model.Offer, error)
	}{
		{endpointCheap, p.cheapRequestURL, p.fetchCheapRaw, inspectCheapRaw, parseCheapResponse},
		{endpointLatest, p.latestRequestURL, p.fetchLatestRaw, inspectLatestRaw, parseLatestResponse},
	}
	for _, ep := range endpoints {
		key := cacheKey(ep.name, q.From, q.To, q.Date, p.currency)
		reqURL := ep.requestURL(q)
		fromCache := false
		var raw []byte
		var httpStatus int
		var err error

		if p.cache != nil {
			if cached, ok, _ := p.cache.Get(p.Name(), key); ok {
				raw, fromCache, httpStatus = cached, true, 200
			}
		}
		if raw == nil {
			fetchFn := func() ([]byte, error) {
				body, status, fetchErr := ep.fetch(ctx, q)
				httpStatus = status
				return body, fetchErr
			}
			if p.cache != nil {
				raw, err = p.cache.Fetch(p.Name(), key, fetchFn)
			} else {
				raw, err = fetchFn()
			}
		}
		if err != nil {
			ed := endpointDebug{
				Endpoint:    ep.name,
				RequestURL:  reqURL,
				FromCache:   fromCache,
				HTTPStatus:  httpStatus,
				EmptyReason: EmptyHTTPError,
				ParseError:  err.Error(),
			}
			rep.Endpoints = append(rep.Endpoints, ed)
			rep.FinalReason = EmptyHTTPError
			return nil, "", rep, err
		}

		stats := ep.inspect(raw, q)
		stats.Endpoint = ep.name
		stats.RequestURL = reqURL
		stats.FromCache = fromCache
		stats.HTTPStatus = httpStatus
		if stats.ResponseBytes == 0 {
			stats.ResponseBytes = len(raw)
		}

		offer, parseErr := ep.parse(raw, q, p.currency)
		if parseErr != nil {
			stats.ParseError = parseErr.Error()
			stats.EmptyReason = EmptyParseEmpty
			rep.Endpoints = append(rep.Endpoints, stats)
			rep.FinalReason = EmptyParseEmpty
			return nil, "", rep, parseErr
		}
		if offer != nil {
			stats.ParsedOffers = 1
			stats.EmptyReason = EmptyNone
			rep.Endpoints = append(rep.Endpoints, stats)
			rep.UsedEndpoint = ep.name
			p.enrichOfferAirlines(ctx, q, offer, ep.name, raw)
			return offer, ep.name, rep, nil
		}
		if stats.EmptyReason == "" {
			stats.EmptyReason = EmptyParseEmpty
		}
		rep.Endpoints = append(rep.Endpoints, stats)
	}
	rep.FinalReason = finalizeEmptyReason(rep.Endpoints)
	return nil, "", rep, nil
}

func (p *Provider) SetAPIRoot(root string)    { p.apiRootOverride = root }
func (p *Provider) SetHTTPClient(c *http.Client) { p.client = c }
func (p *Provider) SetVerbose(v bool)           { p.verbose = v }

func (p *Provider) apiRoot() string {
	if p.apiRootOverride != "" {
		return strings.TrimRight(p.apiRootOverride, "/")
	}
	return "https://" + baseHost
}

func (p *Provider) cheapRequestURL(q model.Query) string {
	params := url.Values{}
	params.Set("origin", q.From)
	params.Set("destination", q.To)
	params.Set("depart_date", q.Date)
	params.Set("currency", p.currency)
	return fmt.Sprintf("%s/v1/prices/cheap?%s", p.apiRoot(), params.Encode())
}

func (p *Provider) latestRequestURL(q model.Query) string {
	params := url.Values{}
	params.Set("origin", q.From)
	params.Set("destination", q.To)
	params.Set("currency", p.currency)
	params.Set("period_type", "month")
	params.Set("beginning_of_period", monthStart(q.Date))
	params.Set("one_way", "true")
	params.Set("limit", "100")
	params.Set("show_to_affiliates", "true")
	params.Set("sorting", "price")
	params.Set("trip_class", "0")
	return fmt.Sprintf("%s/v2/prices/latest?%s", p.apiRoot(), params.Encode())
}

func (p *Provider) fetchCheapRaw(ctx context.Context, q model.Query) ([]byte, int, error) {
	return p.doGET(ctx, p.cheapRequestURL(q))
}

func (p *Provider) fetchLatestRaw(ctx context.Context, q model.Query) ([]byte, int, error) {
	return p.doGET(ctx, p.latestRequestURL(q))
}

func cacheKey(endpoint, origin, dest, date, currency string) string {
	return fmt.Sprintf("%s|%s|%s|%s|%s", endpoint, origin, dest, date, currency)
}

func (p *Provider) doGET(ctx context.Context, rawURL string) ([]byte, int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("X-Access-Token", p.token)

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, resp.StatusCode, fmt.Errorf("travelpayouts_data status %d: %s",
			resp.StatusCode, secrets.Redact(string(body), p.token))
	}
	return body, resp.StatusCode, nil
}

func monthStart(date string) string {
	if len(date) >= 7 {
		return date[:7] + "-01"
	}
	return date
}

func formatMoney(m model.Money) string {
	return fmt.Sprintf("%.2f %s", float64(m.Amount)/100, m.Currency)
}

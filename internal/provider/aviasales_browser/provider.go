package aviasales_browser

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/kneumoin/nepal/internal/cache"
	"github.com/kneumoin/nepal/internal/config"
	"github.com/kneumoin/nepal/internal/model"
	"github.com/kneumoin/nepal/internal/provider"
)

const providerName = "aviasales_browser"

// Provider is an experimental local-only browser collector for Aviasales search pages.
type Provider struct {
	opts    Options
	cache   *cache.Store
	fetcher PageFetcher
	limiter *rateLimiter
}

// New creates the browser provider. fetcher may be nil to use ChromeFetcher.
func New(opts Options, fetcher PageFetcher) *Provider {
	cs, _ := cache.New(config.CacheConfig{
		Enabled:   opts.CacheEnabled,
		TTL:       opts.CacheTTL,
		Directory: opts.CacheDir,
	})
	if fetcher == nil {
		fetcher = &ChromeFetcher{Headful: opts.Headful, Timeout: opts.Timeout, Verbose: opts.Verbose}
	}
	return &Provider{
		opts:    opts,
		cache:   cs,
		fetcher: fetcher,
		limiter: newRateLimiter(opts.RateLimit),
	}
}

func (p *Provider) Name() string { return providerName }

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
	url, err := SearchURL(q.From, q.To, q.Date, q.Passengers)
	if err != nil {
		return nil, err
	}
	key := cacheKey(url)

	cached, err := p.loadCache(key)
	if err == nil && cached != nil {
		return p.offersFromCached(cached, q)
	}
	if p.opts.CacheOnly {
		return nil, fmt.Errorf("cache miss for %s (cache-only mode)", url)
	}

	if err := p.limiter.wait(ctx); err != nil {
		return nil, err
	}

	html, err := p.fetcher.Fetch(ctx, url)
	if err != nil {
		return nil, fmt.Errorf("page fetch failed: %w", err)
	}
	extracted, err := parseHTMLWithPageURL(html, q.From, q.To, q.Date, url)
	if err != nil {
		return nil, err
	}
	page := CachedPage{
		URL:         url,
		CollectedAt: time.Now().UTC(),
		Extracted:   extracted,
	}
	if raw, err := encodeCachedPage(page); err == nil && p.cache != nil {
		_ = p.cache.Put(providerName, key, raw)
	}
	return p.offersFromExtracted(extracted, q)
}

func (p *Provider) loadCache(key string) (*CachedPage, error) {
	if p.cache == nil {
		return nil, fmt.Errorf("cache disabled")
	}
	raw, hit, err := p.cache.Get(providerName, key)
	if err != nil {
		return nil, err
	}
	if !hit || len(raw) == 0 {
		return nil, nil
	}
	return decodeCachedPage(raw)
}

func (p *Provider) offersFromCached(page *CachedPage, q model.Query) ([]model.Offer, error) {
	if p.opts.Verbose {
		fmt.Printf("aviasales_browser: cache hit for %s\n", page.URL)
	}
	return p.offersFromExtracted(page.Extracted, q)
}

func (p *Provider) offersFromExtracted(extracted []ExtractedOffer, q model.Query) ([]model.Offer, error) {
	best := pickBestOffer(extracted)
	if best == nil {
		return nil, nil
	}
	o := mapToOffer(*best, q.From, q.To, q.Date)
	return []model.Offer{o}, nil
}

func cacheKey(url string) string {
	return url
}

// SetFetcher replaces the page fetcher (tests).
func (p *Provider) SetFetcher(f PageFetcher) { p.fetcher = f }

// LoadCachedJSON loads a cached page blob (tests).
func LoadCachedJSON(raw []byte) (*CachedPage, error) {
	var page CachedPage
	if err := json.Unmarshal(raw, &page); err != nil {
		return nil, err
	}
	return &page, nil
}

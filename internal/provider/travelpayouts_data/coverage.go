package travelpayouts_data

import (
	"context"
	"fmt"
	"net/url"

	"github.com/kneumoin/nepal/internal/model"
)

func (p *Provider) AnalyzeLegCoverage(ctx context.Context, q model.Query) (model.RouteCoverageRow, error) {
	if p.token == "" {
		return model.RouteCoverageRow{}, fmt.Errorf("TRAVELPAYOUTS_TOKEN not set")
	}
	target := q.TargetDate
	if target == "" {
		target = q.Date
	}
	window := q.CoverageWindowDays
	if window <= 0 {
		window = 3
	}
	idx, err := p.loadMonthIndex(ctx, q.From, q.To, target)
	if err != nil {
		return model.RouteCoverageRow{}, err
	}
	return coverageRowFromIndex(q, idx, target, window, p.currency), nil
}

func (p *Provider) searchCoverageFallback(ctx context.Context, q model.Query) (*model.Offer, string, error) {
	target := q.TargetDate
	if target == "" {
		target = q.Date
	}
	window := q.CoverageWindowDays
	if window <= 0 {
		window = 3
	}
	idx, err := p.loadMonthIndex(ctx, q.From, q.To, target)
	if err != nil {
		return nil, "", err
	}
	dp, ok := idx.nearestInWindow(target, window, q.NotBeforeDate)
	if !ok {
		return nil, "", nil
	}
	endpoint := dp.Source
	if endpoint == "" {
		endpoint = endpointCalendar
	}
	return offerFromDayPrice(q, dp, target, p.currency), endpoint, nil
}

func (p *Provider) loadMonthIndex(ctx context.Context, from, to, targetDate string) (monthPriceIndex, error) {
	month := monthKey(targetDate)
	calKey := cacheKey(endpointCalendar, from, to, month, p.currency)
	matrixKey := cacheKey(endpointMonthMatrix, from, to, month, p.currency)

	calRaw, err := p.fetchCachedRaw(ctx, calKey, func() ([]byte, error) {
		body, _, fetchErr := p.fetchCalendarRaw(ctx, from, to, month)
		return body, fetchErr
	})
	if err != nil {
		return monthPriceIndex{}, err
	}
	calIdx, err := parseCalendarResponse(calRaw)
	if err != nil {
		return monthPriceIndex{}, err
	}

	matrixRaw, err := p.fetchCachedRaw(ctx, matrixKey, func() ([]byte, error) {
		body, _, fetchErr := p.fetchMonthMatrixRaw(ctx, from, to, monthStart(targetDate))
		return body, fetchErr
	})
	if err != nil {
		return monthPriceIndex{}, err
	}
	matrixIdx, err := parseMonthMatrixResponse(matrixRaw)
	if err != nil {
		return monthPriceIndex{}, err
	}
	return mergeMonthIndexes(calIdx, matrixIdx), nil
}

func (p *Provider) fetchCachedRaw(ctx context.Context, key string, fetch func() ([]byte, error)) ([]byte, error) {
	if p.cache != nil {
		return p.cache.Fetch(p.Name(), key, fetch)
	}
	return fetch()
}

func (p *Provider) calendarRequestURL(from, to, month string) string {
	params := url.Values{}
	params.Set("depart_date", month)
	params.Set("origin", from)
	params.Set("destination", to)
	params.Set("calendar_type", "departure_date")
	params.Set("currency", p.currency)
	return fmt.Sprintf("%s/v1/prices/calendar?%s", p.apiRoot(), params.Encode())
}

func (p *Provider) monthMatrixRequestURL(from, to, monthStartDate string) string {
	params := url.Values{}
	params.Set("origin", from)
	params.Set("destination", to)
	params.Set("currency", p.currency)
	params.Set("show_to_affiliates", "true")
	params.Set("month", monthStartDate)
	return fmt.Sprintf("%s/v2/prices/month-matrix?%s", p.apiRoot(), params.Encode())
}

func (p *Provider) fetchCalendarRaw(ctx context.Context, from, to, month string) ([]byte, int, error) {
	return p.doGET(ctx, p.calendarRequestURL(from, to, month))
}

func (p *Provider) fetchMonthMatrixRaw(ctx context.Context, from, to, monthStartDate string) ([]byte, int, error) {
	return p.doGET(ctx, p.monthMatrixRequestURL(from, to, monthStartDate))
}

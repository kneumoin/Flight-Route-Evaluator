package travelpayouts_data

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/kneumoin/nepal/internal/model"
	"github.com/kneumoin/nepal/internal/secrets"
)

// EmptyReason classifies why a leg search returned no offers.
type EmptyReason string

const (
	EmptyNone          EmptyReason = ""
	EmptyAPIEmpty      EmptyReason = "API_EMPTY"      // success=true but no raw rows for route/date
	EmptyParseEmpty    EmptyReason = "PARSE_EMPTY"    // raw rows exist but parser produced nothing
	EmptyDateFiltered  EmptyReason = "DATE_FILTERED"  // latest rows dropped by exact depart_date match
	EmptyRouteFiltered EmptyReason = "ROUTE_FILTERED" // latest rows dropped by origin/destination match
	EmptyHTTPError     EmptyReason = "HTTP_ERROR"
	EmptyAllEndpoints  EmptyReason = "ALL_ENDPOINTS_EMPTY"
)

type endpointDebug struct {
	Endpoint         string
	RequestURL       string
	FromCache        bool
	HTTPStatus       int
	ResponseBytes    int
	APISuccess       bool
	RawOfferCount    int
	AfterRouteFilter int
	AfterDateFilter  int
	ParsedOffers     int
	EmptyReason      EmptyReason
	ParseError       string
}

type legDebugReport struct {
	Query       model.Query
	Endpoints   []endpointDebug
	FinalReason EmptyReason
	UsedEndpoint string
}

func (p *Provider) logLegDebug(rep legDebugReport) {
	if !p.verbose {
		return
	}
	q := rep.Query
	fmt.Printf("\ntravelpayouts_data DEBUG %s→%s date=%s\n", q.From, q.To, q.Date)
	for _, ep := range rep.Endpoints {
		fmt.Printf("  [%s]\n", ep.Endpoint)
		fmt.Printf("    request_url: %s\n", secrets.Redact(ep.RequestURL, p.token))
		fmt.Printf("    cache_hit: %v\n", ep.FromCache)
		if ep.HTTPStatus > 0 {
			fmt.Printf("    http_status: %d\n", ep.HTTPStatus)
		}
		fmt.Printf("    response_bytes: %d\n", ep.ResponseBytes)
		fmt.Printf("    api_success: %v\n", ep.APISuccess)
		fmt.Printf("    raw_offers: %d\n", ep.RawOfferCount)
		if ep.Endpoint == endpointLatest {
			fmt.Printf("    after_route_filter: %d\n", ep.AfterRouteFilter)
			fmt.Printf("    after_date_filter: %d\n", ep.AfterDateFilter)
		}
		fmt.Printf("    parsed_offers: %d\n", ep.ParsedOffers)
		if ep.ParseError != "" {
			fmt.Printf("    parse_error: %s\n", ep.ParseError)
		}
		if ep.EmptyReason != "" && ep.ParsedOffers == 0 {
			fmt.Printf("    zero_reason: %s\n", ep.EmptyReason)
		}
	}
	if rep.UsedEndpoint != "" {
		fmt.Printf("  RESULT: offer via %s\n", rep.UsedEndpoint)
	} else {
		fmt.Printf("  RESULT: no offers (%s)\n", rep.FinalReason)
	}
}

func inspectCheapRaw(raw []byte, q model.Query) endpointDebug {
	d := endpointDebug{Endpoint: endpointCheap, ResponseBytes: len(raw)}
	var payload struct {
		Success bool                              `json:"success"`
		Data    map[string]map[string]cheapEntry `json:"data"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		d.ParseError = err.Error()
		d.EmptyReason = EmptyParseEmpty
		return d
	}
	d.APISuccess = payload.Success
	destMap, ok := payload.Data[strings.ToUpper(q.To)]
	if !ok || len(destMap) == 0 {
		d.RawOfferCount = 0
		d.EmptyReason = EmptyAPIEmpty
		return d
	}
	d.RawOfferCount = len(destMap)
	var best *cheapEntry
	for _, e := range destMap {
		cp := e
		if best == nil || cp.Price < best.Price {
			best = &cp
		}
	}
	if best == nil || best.Price <= 0 {
		d.EmptyReason = EmptyParseEmpty
		return d
	}
	d.ParsedOffers = 1
	return d
}

func inspectLatestRaw(raw []byte, q model.Query) endpointDebug {
	d := endpointDebug{Endpoint: endpointLatest, ResponseBytes: len(raw)}
	var payload struct {
		Success bool          `json:"success"`
		Data    []latestEntry `json:"data"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		d.ParseError = err.Error()
		d.EmptyReason = EmptyParseEmpty
		return d
	}
	d.APISuccess = payload.Success
	d.RawOfferCount = len(payload.Data)
	if d.RawOfferCount == 0 {
		d.EmptyReason = EmptyAPIEmpty
		return d
	}

	routeMatch := 0
	dateMatch := 0
	var best *latestEntry
	for i := range payload.Data {
		e := &payload.Data[i]
		if !strings.EqualFold(e.Origin, q.From) || !strings.EqualFold(e.Destination, q.To) {
			continue
		}
		routeMatch++
		if e.DepartDate != "" && e.DepartDate != q.Date {
			continue
		}
		dateMatch++
		if best == nil || e.Value < best.Value {
			best = e
		}
	}
	d.AfterRouteFilter = routeMatch
	d.AfterDateFilter = dateMatch

	if routeMatch == 0 {
		d.EmptyReason = EmptyRouteFiltered
		return d
	}
	if dateMatch == 0 {
		d.EmptyReason = EmptyDateFiltered
		return d
	}
	if best == nil || best.Value <= 0 {
		d.EmptyReason = EmptyParseEmpty
		return d
	}
	d.ParsedOffers = 1
	return d
}

func finalizeEmptyReason(endpoints []endpointDebug) EmptyReason {
	if len(endpoints) == 0 {
		return EmptyAllEndpoints
	}
	// Prefer the most informative reason from the last endpoint tried.
	for i := len(endpoints) - 1; i >= 0; i-- {
		ep := endpoints[i]
		if ep.EmptyReason != "" {
			return ep.EmptyReason
		}
	}
	return EmptyAllEndpoints
}

package travelpayouts_data

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/kneumoin/nepal/internal/model"
)

func (p *Provider) enrichOfferAirlines(ctx context.Context, q model.Query, offer *model.Offer, endpoint string, raw []byte) {
	target := q.TargetDate
	if target == "" {
		target = q.Date
	}
	var airlines []string
	if endpoint == endpointCheap && len(raw) > 0 {
		airlines = airlinesFromCheapRaw(raw, q.To)
	}
	if endpoint != endpointCheap || len(airlines) == 0 {
		if idx, err := p.loadMonthIndex(ctx, q.From, q.To, target); err == nil {
			airlines = model.MergeAirlineLists(airlines, idx.airlinesOnDate(target))
		}
	}
	if len(offer.Segments) > 0 {
		airlines = model.MergeAirlineLists(airlines, []string{offer.Segments[0].Airline})
	}
	offer.AvailableAirlines = airlines
}

func (p *Provider) enrichOfferAirlinesFromIndex(ctx context.Context, q model.Query, offer *model.Offer) {
	target := q.TargetDate
	if target == "" {
		target = q.Date
	}
	var airlines []string
	if idx, err := p.loadMonthIndex(ctx, q.From, q.To, target); err == nil {
		airlines = model.MergeAirlineLists(airlines, idx.airlinesOnDate(target))
	}
	if len(offer.Segments) > 0 {
		airlines = model.MergeAirlineLists(airlines, []string{offer.Segments[0].Airline})
	}
	offer.AvailableAirlines = airlines
}

func airlinesFromCheapRaw(raw []byte, dest string) []string {
	var payload struct {
		Data map[string]map[string]cheapEntry `json:"data"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil
	}
	destMap, ok := payload.Data[strings.ToUpper(dest)]
	if !ok {
		return nil
	}
	var codes []string
	for _, e := range destMap {
		if a := strings.ToUpper(strings.TrimSpace(e.Airline)); a != "" {
			codes = append(codes, a)
		}
	}
	return model.MergeAirlines(codes...)
}

func (idx monthPriceIndex) airlinesOnDate(date string) []string {
	if e, ok := idx.ByDate[date]; ok {
		if a := strings.ToUpper(strings.TrimSpace(e.Airline)); a != "" {
			return []string{a}
		}
	}
	return nil
}

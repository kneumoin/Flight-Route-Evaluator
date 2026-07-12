package report

import (
	"fmt"
	"html"
	"os"
	"path/filepath"
	"strings"

	"github.com/kneumoin/nepal/internal/links"
	"github.com/kneumoin/nepal/internal/model"
)

// DefaultFinalists is how many top branches land in the "for dummies" document.
const DefaultFinalists = 3

// FinalistCard is one big, human-readable card in finalists.html.
type FinalistCard struct {
	Rank         int
	BranchName   string
	Path         string   // MOW → DXB → KTM
	OutboundDate string   // 26.09.2026
	ReturnDate   string   // 11.11.2026 (empty for one-way)
	Price        string   // ≈ $850 (touда+обратно)
	Airlines     []string // carriers seen in the snapshot
	AviasalesURL string
}

// WriteFinalistsHTML writes a simple top-N booking document.
// Branches must already be ranked (best first); only OK/Partial ones are used.
func WriteFinalistsHTML(path string, result *model.EvaluationResult, topN int) error {
	if topN <= 0 {
		topN = DefaultFinalists
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(renderFinalistsHTML(result, topN)), 0o644)
}

// RenderFinalistsHTMLForTest exposes the finalists HTML for unit tests.
func RenderFinalistsHTMLForTest(result *model.EvaluationResult, topN int) string {
	return renderFinalistsHTML(result, topN)
}

func renderFinalistsHTML(result *model.EvaluationResult, topN int) string {
	cards := finalistCards(result, topN)
	snapshot := result.GeneratedAt.UTC().Format("02.01.2006 15:04 UTC")

	var b strings.Builder
	b.WriteString("<!DOCTYPE html><html lang=\"ru\"><head><meta charset=\"utf-8\">")
	b.WriteString("<meta name=\"viewport\" content=\"width=device-width, initial-scale=1\">")
	b.WriteString("<title>Топ маршрутов — MOW ⇄ KTM</title><style>")
	b.WriteString(`
body{font-family:system-ui,-apple-system,sans-serif;margin:0;padding:1.5rem;background:#f6f7f9;color:#111}
.wrap{max-width:760px;margin:0 auto}
h1{margin:0 0 .25rem;font-size:1.6rem}
.meta{color:#555;margin:0 0 1rem}
.help{background:#eef6ff;border:1px solid #bcdcff;border-radius:10px;padding:.9rem 1.1rem;margin:0 0 1.5rem;font-size:.95rem;line-height:1.5}
.help ol{margin:.4rem 0 0;padding-left:1.2rem}
.card{background:#fff;border:1px solid #e3e6ea;border-radius:14px;padding:1.2rem 1.3rem;margin:0 0 1.1rem;box-shadow:0 1px 3px rgba(0,0,0,.05)}
.card .top{display:flex;align-items:center;gap:.7rem;flex-wrap:wrap}
.rank{display:inline-flex;align-items:center;justify-content:center;width:2rem;height:2rem;border-radius:50%;background:#0b57d0;color:#fff;font-weight:700;font-size:1rem}
.name{font-size:1.25rem;font-weight:700}
.path{font-size:1.05rem;color:#111;margin:.7rem 0 .2rem;letter-spacing:.02em}
.dates{color:#444;margin:.15rem 0 .6rem}
.price{font-size:1.7rem;font-weight:800;color:#0a7a2f;margin:.3rem 0}
.airlines{color:#333;margin:.2rem 0 .9rem;font-size:.95rem}
.btn{display:inline-block;background:#ff6f00;color:#fff;text-decoration:none;font-weight:700;padding:.7rem 1.3rem;border-radius:10px;font-size:1.05rem}
.btn:hover{background:#e56400}
.empty{background:#fff;border:1px solid #e3e6ea;border-radius:14px;padding:1.5rem;text-align:center;color:#666}
.foot{color:#888;font-size:.8rem;margin-top:1.5rem;text-align:center}
`)
	b.WriteString("</style></head><body><div class=\"wrap\">")

	b.WriteString("<h1>Куда лететь: топ вариантов</h1>")
	b.WriteString(fmt.Sprintf("<p class=\"meta\">%s · цены на момент: <strong>%s</strong></p>",
		html.EscapeString(formatTripTitle(result.Trip)), html.EscapeString(snapshot)))

	b.WriteString(`<div class="help"><strong>Как пользоваться:</strong>
<ol>
<li>Цена ниже — <em>ориентир</em> из кэша, не финальная.</li>
<li>Нажми оранжевую кнопку — откроется поиск на Aviasales с уже подставленными городами и датами.</li>
<li>Там смотришь реальную цену и покупаешь билет.</li>
</ol></div>`)

	if len(cards) == 0 {
		b.WriteString(`<div class="empty">Нет подходящих маршрутов с данными.</div>`)
	} else {
		for _, c := range cards {
			b.WriteString(`<div class="card">`)
			b.WriteString(`<div class="top">`)
			b.WriteString(fmt.Sprintf(`<span class="rank">%d</span>`, c.Rank))
			b.WriteString(`<span class="name">` + html.EscapeString(c.BranchName) + `</span>`)
			b.WriteString(`</div>`)

			b.WriteString(`<div class="path">` + html.EscapeString(c.Path) + `</div>`)

			dates := "Туда: " + html.EscapeString(c.OutboundDate)
			if c.ReturnDate != "" {
				dates += " · Обратно: " + html.EscapeString(c.ReturnDate)
			}
			b.WriteString(`<div class="dates">` + dates + `</div>`)

			b.WriteString(`<div class="price">` + html.EscapeString(c.Price) + `</div>`)

			if len(c.Airlines) > 0 {
				b.WriteString(`<div class="airlines">Авиакомпании: ` +
					html.EscapeString(strings.Join(c.Airlines, ", ")) + `</div>`)
			}

			if c.AviasalesURL != "" {
				b.WriteString(fmt.Sprintf(`<a class="btn" href="%s" target="_blank" rel="noopener">Посмотреть билеты на Aviasales →</a>`,
					html.EscapeString(c.AviasalesURL)))
			}
			b.WriteString(`</div>`)
		}
	}

	b.WriteString(`<p class="foot">Данные: кэшированный Travelpayouts Data API. Итоговая цена и наличие — только на Aviasales.</p>`)
	b.WriteString("</div></body></html>")
	return b.String()
}

func finalistCards(result *model.EvaluationResult, topN int) []FinalistCard {
	passengers := result.Trip.Passengers
	if passengers < 1 {
		passengers = 1
	}

	var cards []FinalistCard
	rank := 0
	for _, br := range result.Branches {
		if br.Status != model.StatusOK && br.Status != model.StatusPartial {
			continue
		}
		rank++
		if rank > topN {
			break
		}

		card := FinalistCard{
			Rank:         rank,
			BranchName:   br.BranchName,
			Path:         finalistPath(result.Trip, br),
			OutboundDate: humanDate(finalistOutboundDate(result.Trip, br)),
			Airlines:     finalistAirlines(br),
		}

		total, cur := finalistTotal(br)
		card.Price = formatFinalistPrice(total, cur)

		if result.Trip.ReturnDate != "" {
			ret := finalistReturnDate(result.Trip, br)
			card.ReturnDate = humanDate(ret)
			if u, err := links.RoundTripShortURL(result.Trip.Origin, result.Trip.Destination,
				finalistOutboundDate(result.Trip, br), ret, passengers); err == nil {
				card.AviasalesURL = u
			}
		} else {
			if u, err := links.OneWaySearchURL(result.Trip.Origin, result.Trip.Destination,
				finalistOutboundDate(result.Trip, br), passengers); err == nil {
				card.AviasalesURL = u
			}
		}

		cards = append(cards, card)
	}
	return cards
}

// finalistPath renders MOW → HUB → KTM from the outbound offer legs (fallback to endpoints).
func finalistPath(trip model.TripMeta, br model.BranchResult) string {
	o := outboundOffer(br)
	if o != nil && len(o.LegDetails) > 0 {
		points := []string{o.LegDetails[0].From}
		for _, leg := range o.LegDetails {
			points = append(points, leg.To)
		}
		return strings.Join(points, " → ")
	}
	return trip.Origin + " → " + trip.Destination
}

func finalistOutboundDate(trip model.TripMeta, br model.BranchResult) string {
	o := outboundOffer(br)
	if o != nil && len(o.LegDetails) > 0 && o.LegDetails[0].SearchDate != "" {
		return o.LegDetails[0].SearchDate
	}
	return trip.DepartureDate
}

func finalistReturnDate(trip model.TripMeta, br model.BranchResult) string {
	if br.ReturnOffer != nil && len(br.ReturnOffer.LegDetails) > 0 && br.ReturnOffer.LegDetails[0].SearchDate != "" {
		return br.ReturnOffer.LegDetails[0].SearchDate
	}
	if trip.ReturnDate != "" {
		return trip.ReturnDate
	}
	return trip.DepartureDate
}

func finalistAirlines(br model.BranchResult) []string {
	seen := map[string]bool{}
	var out []string
	add := func(names []string) {
		for _, n := range names {
			n = strings.TrimSpace(n)
			if n == "" || seen[n] {
				continue
			}
			seen[n] = true
			out = append(out, n)
		}
	}
	for _, o := range []*model.Offer{outboundOffer(br), br.ReturnOffer} {
		if o == nil {
			continue
		}
		for _, leg := range o.LegDetails {
			if leg.Airline != "" {
				add([]string{leg.Airline})
			} else {
				add(leg.AvailableAirlines)
			}
		}
	}
	return out
}

// finalistTotal sums outbound + return (or single offer) in their original currency.
func finalistTotal(br model.BranchResult) (int64, string) {
	var total int64
	var cur string
	for _, o := range []*model.Offer{outboundOffer(br), br.ReturnOffer} {
		if o == nil || o.Price.Amount <= 0 {
			continue
		}
		total += o.Price.Amount
		cur = o.Price.Currency
	}
	if total == 0 && br.Offer != nil {
		return br.Offer.Price.Amount, br.Offer.Price.Currency
	}
	return total, cur
}

func outboundOffer(br model.BranchResult) *model.Offer {
	if br.OutboundOffer != nil {
		return br.OutboundOffer
	}
	return br.Offer
}

func formatFinalistPrice(amount int64, currency string) string {
	if amount <= 0 {
		return "цена — на Aviasales"
	}
	return "≈ " + formatMoneyAmount(amount, currency)
}

func humanDate(iso string) string {
	if len(iso) != 10 {
		return iso
	}
	return iso[8:10] + "." + iso[5:7] + "." + iso[0:4]
}

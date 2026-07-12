package report

import (
	"encoding/csv"
	"fmt"
	"html"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/kneumoin/nepal/internal/links"
	"github.com/kneumoin/nepal/internal/model"
)

// BookingRow is one line in the booking sheet: date, price, airline snapshot, Aviasales link.
type BookingRow struct {
	BranchID     string
	BranchName   string
	Direction    string
	Date         string
	Route        string
	Price        string
	Airline      string
	DataLabel    string
	AviasalesURL string
	LinkKind     string
}

func WriteBookingHTML(path string, result *model.EvaluationResult) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(renderBookingHTML(result)), 0o644)
}

// RenderBookingHTMLForTest exposes booking HTML for unit tests.
func RenderBookingHTMLForTest(result *model.EvaluationResult) string {
	return renderBookingHTML(result)
}

func WriteBookingCSV(path string, result *model.EvaluationResult) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	w := csv.NewWriter(f)
	_ = w.Write([]string{"branch", "direction", "date", "route", "price", "airline", "data_snapshot", "link_kind", "aviasales_url"})
	for _, row := range bookingRows(result) {
		_ = w.Write([]string{
			row.BranchName, row.Direction, row.Date, row.Route, row.Price,
			row.Airline, row.DataLabel, row.LinkKind, row.AviasalesURL,
		})
	}
	w.Flush()
	return w.Error()
}

func renderBookingHTML(result *model.EvaluationResult) string {
	ts := result.GeneratedAt.UTC().Format("02.01.2006 15:04 UTC")
	rows := bookingRows(result)

	var b strings.Builder
	b.WriteString("<!DOCTYPE html><html lang=\"ru\"><head><meta charset=\"utf-8\">")
	b.WriteString("<title>Лист покупки — MOW ⇄ KTM</title><style>")
	b.WriteString(`body{font-family:system-ui,sans-serif;margin:0;padding:1rem 2rem;background:#fff;color:#111}
h1{margin:0 0 .25rem} .meta{color:#555;margin-bottom:1.5rem}
table{width:100%;border-collapse:collapse;font-size:.95rem}
th,td{border:1px solid #ddd;padding:.55rem .65rem;text-align:left;vertical-align:top}
th{background:#f3f4f6}
a{color:#0b57d0}
.badge{display:inline-block;background:#eef2ff;color:#3730a3;border-radius:4px;padding:.1rem .45rem;font-size:.8rem;margin-left:.35rem}
.note{background:#fffbeb;border:1px solid #fcd34d;border-radius:8px;padding:.75rem 1rem;margin:1rem 0;font-size:.9rem}`)
	b.WriteString("</style></head><body>")

	b.WriteString("<h1>Лист для покупки</h1>")
	b.WriteString(fmt.Sprintf("<p class=\"meta\">%s · снимок данных: <strong>%s</strong></p>", html.EscapeString(formatTripTitle(result.Trip)), html.EscapeString(ts)))
	b.WriteString(`<p class="note">Цены из кэшированного Travelpayouts Data API — не live-поиск. Ссылки открывают <strong>поиск на Aviasales</strong> (не конкретный билет). Для точной цены и покупки проверяйте на сайте.</p>`)

	b.WriteString("<table><thead><tr>")
	for _, h := range []string{"Маршрут", "Направление", "Дата", "Плечо", "Сумма", "Авиакомпания", "Aviasales"} {
		b.WriteString("<th>" + h + "</th>")
	}
	b.WriteString("</tr></thead><tbody>")

	if len(rows) == 0 {
		b.WriteString("<tr><td colspan=\"7\">Нет маршрутов с данными</td></tr>")
	} else {
		for _, row := range rows {
			b.WriteString("<tr>")
			b.WriteString("<td>" + html.EscapeString(row.BranchName) + "</td>")
			b.WriteString("<td>" + html.EscapeString(row.Direction) + "</td>")
			b.WriteString("<td>" + html.EscapeString(row.Date) + "</td>")
			b.WriteString("<td>" + html.EscapeString(row.Route) + "</td>")
			b.WriteString("<td>" + html.EscapeString(row.Price) + "</td>")
			airlineCell := html.EscapeString(row.Airline)
			if row.DataLabel != "" {
				airlineCell += `<span class="badge">` + html.EscapeString(row.DataLabel) + `</span>`
			}
			b.WriteString("<td>" + airlineCell + "</td>")
			if row.AviasalesURL != "" {
				label := "Поиск"
				if row.LinkKind == "round_trip" {
					label = "RT поиск"
				} else if row.LinkKind == "leg" {
					label = "Leg"
				}
				b.WriteString(fmt.Sprintf(`<td><a href="%s" target="_blank" rel="noopener">%s</a></td>`,
					html.EscapeString(row.AviasalesURL), label))
			} else {
				b.WriteString("<td></td>")
			}
			b.WriteString("</tr>")
		}
	}
	b.WriteString("</tbody></table></body></html>")
	return b.String()
}

func bookingRows(result *model.EvaluationResult) []BookingRow {
	snapshot := formatDataSnapshot(result.GeneratedAt)
	passengers := result.Trip.Passengers
	if passengers < 1 {
		passengers = 1
	}

	var rows []BookingRow
	for _, br := range result.Branches {
		if br.Status != model.StatusOK && br.Status != model.StatusPartial {
			continue
		}

		directions := directionOffers(br)

		var total int64
		var totalCur string
		for _, dir := range directions {
			if dir.offer == nil {
				continue
			}
			for _, leg := range dir.offer.LegDetails {
				rows = append(rows, legBookingRow(br, dir.label, leg, snapshot, passengers))
			}
			if dir.offer.Price.Amount > 0 {
				total += dir.offer.Price.Amount
				totalCur = dir.offer.Price.Currency
			}
		}

		if br.ReturnOffer != nil && total > 0 {
			dep := result.Trip.DepartureDate
			ret := pickReturnDate(result.Trip, br.ReturnOffer)
			if u, err := links.RoundTripShortURL(result.Trip.Origin, result.Trip.Destination, dep, ret, passengers); err == nil {
				rows = append(rows, BookingRow{
					BranchID:     br.BranchID,
					BranchName:   br.BranchName,
					Direction:    "итого RT",
					Date:         dep + " / " + ret,
					Route:        result.Trip.Origin + " ⇄ " + result.Trip.Destination,
					Price:        formatMoneyAmount(total, totalCur),
					Airline:      "—",
					DataLabel:    snapshot,
					AviasalesURL: u,
					LinkKind:     "round_trip",
				})
			}
		} else if br.Offer != nil && br.ReturnOffer == nil && br.OutboundOffer == nil {
			// one-way: add direct search link on first leg date
			if len(br.Offer.LegDetails) > 0 {
				leg := br.Offer.LegDetails[0]
				if u, err := links.OneWaySearchURL(result.Trip.Origin, result.Trip.Destination, legDateOrTrip(leg, result.Trip.DepartureDate), passengers); err == nil {
					rows = append(rows, BookingRow{
						BranchID:     br.BranchID,
						BranchName:   br.BranchName,
						Direction:    "поиск OW",
						Date:         legDateOrTrip(leg, result.Trip.DepartureDate),
						Route:        result.Trip.Origin + " → " + result.Trip.Destination,
						Price:        formatBranchTotal(br.Offer),
						Airline:      "—",
						DataLabel:    snapshot,
						AviasalesURL: u,
						LinkKind:     "round_trip",
					})
				}
			}
		}
	}
	return rows
}

func legBookingRow(br model.BranchResult, direction string, leg model.LegDetail, snapshot string, passengers int) BookingRow {
	date := leg.SearchDate
	if date == "" && !leg.Departure.IsZero() {
		date = leg.Departure.Format("2006-01-02")
	}
	airline := leg.Airline
	if airline == "" {
		if len(leg.AvailableAirlines) > 0 {
			airline = strings.Join(leg.AvailableAirlines, ", ")
		} else {
			airline = "—"
		}
	}
	if leg.EstimatedDate {
		snapshot = snapshot + " · дата оценочная"
	}

	u, _ := links.OneWaySearchURL(leg.From, leg.To, date, passengers)
	return BookingRow{
		BranchID:     br.BranchID,
		BranchName:   br.BranchName,
		Direction:    direction,
		Date:         date,
		Route:        leg.From + "→" + leg.To,
		Price:        formatMoneyAmount(leg.Price.Amount, leg.Price.Currency),
		Airline:      airline,
		DataLabel:    snapshot,
		AviasalesURL: u,
		LinkKind:     "leg",
	}
}

func formatDataSnapshot(at time.Time) string {
	return "кэш API · " + at.UTC().Format("02.01.2006 15:04 UTC")
}

func formatMoneyAmount(amount int64, currency string) string {
	if amount == 0 {
		return "—"
	}
	if currency == "" {
		currency = "USD"
	}
	if currency == "USD" {
		return fmt.Sprintf("$%.0f", float64(amount)/100)
	}
	return fmt.Sprintf("%.0f %s", float64(amount)/100, currency)
}

func formatBranchTotal(o *model.Offer) string {
	if o == nil {
		return "—"
	}
	if o.PriceNormalized != nil {
		return formatMoneyAmount(o.PriceNormalized.Amount, o.PriceNormalized.Currency)
	}
	return formatMoneyAmount(o.Price.Amount, o.Price.Currency)
}

func directionOffers(br model.BranchResult) []struct {
	label string
	offer *model.Offer
} {
	if br.OutboundOffer != nil || br.ReturnOffer != nil {
		return []struct {
			label string
			offer *model.Offer
		}{
			{"туда", br.OutboundOffer},
			{"обратно", br.ReturnOffer},
		}
	}
	if br.Offer != nil {
		return []struct {
			label string
			offer *model.Offer
		}{{"туда", br.Offer}}
	}
	return nil
}

func pickReturnDate(trip model.TripMeta, ret *model.Offer) string {
	if ret != nil && len(ret.LegDetails) > 0 && ret.LegDetails[0].SearchDate != "" {
		return ret.LegDetails[0].SearchDate
	}
	if trip.ReturnDate != "" {
		return trip.ReturnDate
	}
	return trip.DepartureDate
}

func legDateOrTrip(leg model.LegDetail, tripDate string) string {
	if leg.SearchDate != "" {
		return leg.SearchDate
	}
	return tripDate
}

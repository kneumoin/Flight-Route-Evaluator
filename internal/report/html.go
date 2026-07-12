package report

import (
	"encoding/json"
	"fmt"
	"html"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/kneumoin/nepal/internal/model"
)

func WriteHTML(path string, result *model.EvaluationResult) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	content := renderHTML(result)
	return os.WriteFile(path, []byte(content), 0o644)
}

// RenderHTMLForTest exposes HTML rendering for unit tests.
func RenderHTMLForTest(result *model.EvaluationResult) string {
	return renderHTML(result)
}

func renderHTML(result *model.EvaluationResult) string {
	dataJSON, _ := json.Marshal(result)
	ts := result.GeneratedAt.UTC().Format(time.RFC3339)

	var b strings.Builder
	b.WriteString("<!DOCTYPE html><html lang=\"ru\"><head><meta charset=\"utf-8\">")
	b.WriteString("<title>MOW → KTM</title><style>")
	b.WriteString(`body{font-family:system-ui,sans-serif;margin:0;padding:1rem 2rem;background:#f6f7f9;color:#1a1a1a}
header{display:flex;justify-content:space-between;align-items:flex-start;border-bottom:2px solid #333;padding-bottom:1rem;margin-bottom:1.5rem}
.lang-switch button{padding:.4rem .8rem;margin-left:.25rem;cursor:pointer;border:1px solid #888;background:#fff;border-radius:4px}
.lang-switch button.active{background:#333;color:#fff}
table{width:100%;border-collapse:collapse;margin:1rem 0}
th,td{border:1px solid #ccc;padding:.5rem;text-align:left}
th{background:#eee}
.badge{display:inline-block;padding:.2rem .5rem;border-radius:4px;font-size:.85rem;margin-right:.25rem}
.badge-low{background:#d4edda}.badge-med{background:#fff3cd}.badge-high{background:#f8d7da}
.visa-warn-bold{font-weight:bold;color:#a00}
.regional-warn-bold{font-weight:bold;color:#b45309;background:#fff3cd;padding:.25rem .5rem;border-radius:4px}
.risk-note{font-size:.9rem;color:#555}
details{margin:.75rem 0;border:1px solid #ddd;border-radius:6px;padding:.5rem 1rem;background:#fff}
.recommendation{background:#e8f4fd;padding:1rem;border-radius:8px;margin-top:1.5rem}`)
	b.WriteString("</style></head><body>")

	b.WriteString("<header><div>")
	b.WriteString(fmt.Sprintf("<h1><span data-i18n=\"title\">%s</span></h1>", html.EscapeString(formatTripTitle(result.Trip))))
	b.WriteString(fmt.Sprintf("<p><span data-i18n=\"generated\">Сгенерировано</span>: %s</p>", html.EscapeString(ts)))
	b.WriteString("</div><div class=\"lang-switch\">")
	b.WriteString("<button id=\"btn-ru\" class=\"active\" onclick=\"setLang('ru')\">RU</button>")
	b.WriteString("<button id=\"btn-en\" onclick=\"setLang('en')\">EN</button></div></header>")

	b.WriteString("<h2 data-i18n=\"summary\">Сводный рейтинг</h2>")
	b.WriteString("<table id=\"summary-table\"><thead><tr>")
	for _, k := range []string{"rank", "route", "status", "price", "airlines", "visa", "operational_disruption", "score"} {
		b.WriteString(fmt.Sprintf("<th data-i18n=\"col_%s\">%s</th>", k, k))
	}
	b.WriteString("</tr></thead><tbody>")
	rank := 1
	for _, br := range result.Branches {
		if br.Status != model.StatusOK && br.Status != model.StatusPartial {
			continue
		}
		b.WriteString("<tr>")
		b.WriteString(fmt.Sprintf("<td>%d</td><td>%s</td><td>%s</td>", rank, html.EscapeString(br.BranchName), br.Status))
		b.WriteString(fmt.Sprintf("<td>%s</td>", html.EscapeString(formatBranchPrice(br, "ru"))))
		b.WriteString(fmt.Sprintf("<td>%s</td>", html.EscapeString(formatBranchAirlines(br))))
		b.WriteString(fmt.Sprintf("<td>%s</td>", html.EscapeString(string(br.VisaCategory))))
		b.WriteString(fmt.Sprintf("<td>%s</td>", html.EscapeString(string(br.OperationalDisruptionRisk))))
		if br.Score != nil {
			b.WriteString(fmt.Sprintf("<td>%.1f</td>", *br.Score))
		} else {
			b.WriteString("<td></td>")
		}
		b.WriteString("</tr>")
		rank++
	}
	b.WriteString("</tbody></table>")

	b.WriteString("<h2 data-i18n=\"coverage\">Покрытие маршрутов (cached API)</h2>")
	b.WriteString(renderCoverageTable(result.RouteCoverage))

	b.WriteString("<h2 data-i18n=\"details\">Детали маршрутов</h2>")
	for _, br := range result.Branches {
		b.WriteString("<details><summary>")
		b.WriteString(html.EscapeString(br.BranchName))
		if br.Score != nil {
			b.WriteString(fmt.Sprintf(" — %.1f", *br.Score))
		}
		b.WriteString("</summary>")
		b.WriteString(fmt.Sprintf("<p><span data-i18n=\"status\">Статус</span>: %s</p>", br.Status))
		b.WriteString(renderVisaWarning(br.VisaCategory))
		b.WriteString(renderOperationalDisruptionWarning(br))
		b.WriteString(renderBranchAirlines(br))
		if br.PriceComparison != nil {
			b.WriteString(fmt.Sprintf("<p><span data-i18n=\"price_compare\">Сравнение цен</span>: %s</p>",
				html.EscapeString(formatBranchPrice(br, "ru"))))
		}
		b.WriteString(renderPartialStatusNotes(br))
		if br.Status == model.StatusPartial && br.Offer == nil {
			b.WriteString(renderBranchCoverageForBranch(result.RouteCoverage, br.BranchID))
		}
		if len(br.ReasonCodes) > 0 {
			b.WriteString("<p><span data-i18n=\"reasons\">Причины</span>: ")
			for _, c := range br.ReasonCodes {
				b.WriteString(fmt.Sprintf("<code>%s</code> ", c))
			}
			b.WriteString("</p>")
		}
		if br.Breakdown != nil {
			bd := br.Breakdown
			b.WriteString(fmt.Sprintf("<p data-i18n=\"breakdown\">Разбор: price=%.0f duration=%.0f baggage=%.0f visa=%.0f self=%.0f late=%.0f</p>",
				bd.Price, bd.Duration, bd.Baggage, bd.Visa, bd.SelfTransfer, bd.LateArrival))
		}
		if br.Offer != nil {
			o := br.Offer
			b.WriteString(renderRiskBadges(o))
			b.WriteString(renderConnectionStatus(o))
			b.WriteString(renderDataQualityNotes(o))
			b.WriteString(renderLegTable(o))
			for _, seg := range o.Segments {
				airline := seg.Airline
				if airline != "" && seg.FlightNumber != "" && !strings.HasPrefix(seg.FlightNumber, airline) {
					b.WriteString(fmt.Sprintf("<p>%s %s %s→%s %s %s</p>",
						html.EscapeString(airline), html.EscapeString(seg.FlightNumber), seg.From, seg.To,
						seg.Departure.Format("2006-01-02 15:04 MST"), seg.Arrival.Format("2006-01-02 15:04 MST")))
				} else {
					b.WriteString(fmt.Sprintf("<p>%s %s→%s %s %s</p>",
						html.EscapeString(seg.FlightNumber), seg.From, seg.To,
						seg.Departure.Format("2006-01-02 15:04 MST"), seg.Arrival.Format("2006-01-02 15:04 MST")))
				}
			}
		}
		b.WriteString("</details>")
	}

	b.WriteString("<h2 data-i18n=\"unavailable\">Недоступные / отклонённые</h2><ul>")
	for _, br := range result.Branches {
		if br.Status == model.StatusOK || br.Status == model.StatusPartial {
			continue
		}
		b.WriteString(fmt.Sprintf("<li><strong>%s</strong> (%s): ", html.EscapeString(br.BranchName), br.Status))
		for _, c := range br.ReasonCodes {
			b.WriteString(fmt.Sprintf("<span data-reason=\"%s\">%s</span> ", c, html.EscapeString(model.ReasonLabel(c, "ru"))))
		}
		b.WriteString("</li>")
	}
	b.WriteString("</ul>")

	top := topBranch(result)
	b.WriteString("<div class=\"recommendation\"><h2 data-i18n=\"recommendation\">Рекомендация</h2>")
	if top != nil {
		b.WriteString(fmt.Sprintf("<p data-i18n=\"top_pick\">Лучший вариант</p><p><strong>%s</strong>", html.EscapeString(top.BranchName)))
		if top.Score != nil {
			b.WriteString(fmt.Sprintf(" (%.1f)", *top.Score))
		}
		b.WriteString("</p>")
	} else {
		b.WriteString("<p data-i18n=\"no_ok\">Нет доступных маршрутов</p>")
	}
	b.WriteString("</div>")

	b.WriteString("<script type=\"application/json\" id=\"eval-data\">")
	b.WriteString(string(dataJSON))
	b.WriteString("</script>")

	b.WriteString(`<script>
const i18n={
ru:{title:"Маршрут",generated:"Сгенерировано",summary:"Сводный рейтинг",details:"Детали маршрутов",
status:"Статус",reasons:"Причины",breakdown:"Разбор оценки",unavailable:"Недоступные / отклонённые",
recommendation:"Рекомендация",top_pick:"Лучший вариант",no_ok:"Нет доступных маршрутов",
col_rank:"#",col_route:"Маршрут",col_status:"Статус",col_price:"Цена",col_airlines:"Авиакомпании",col_duration:"Длительность",col_score:"Оценка",col_visa:"Визовый риск",col_operational_disruption:"Риск операционных сбоев",
visa_required:"Требуется транзитная виза",visa_may_required:"Может потребоваться транзитная виза",
visa_check:"Проверьте правила транзита перед покупкой",visa_risk_label:"Визовый риск",price_compare:"Сравнение цен",available_airlines:"Доступные авиакомпании",
ops_disruption_label:"Риск операционных сбоев",ops_elevated:"Возможны рекомендации авиакомпаниям обходить регион",ops_high:"Возможны закрытие воздушного пространства, аэропорта или массовые отмены",ops_check:"Проверьте текущую обстановку перед покупкой",ops_disclaimer:"Риск срыва логистики для пассажира (отмены, задержки), а не близость к зоне конфликта",
note_cached:"Кэшированные данные о цене",note_baggage:"Багаж неизвестен",note_schedule:"Детали расписания могут быть неполными",bag_unknown:"Багаж неизвестен",
note_browser:"Данные собраны через браузер",note_browser_source:"Собрано с видимой страницы поиска Aviasales",note_browser_unstable:"Может быть неполным или нестабильным",
note_connection_verified:"Стыковка проверена",note_connection_unverified:"Время стыковки не полностью проверено",
note_no_price_target:"Нет цены на целевую дату",note_price_nearby:"Цена с ближайшей даты (кэш)",note_route_incomplete:"Неполные данные маршрута",
status_partial:"частично",
col_leg:"Плечо",col_airline:"Авиакомпания",col_available_airlines:"Доступные АК",col_flight:"Рейс",col_date:"Дата",col_leg_price:"Цена",
col_price_d1:"D-1",col_price_d:"D",col_price_d1p:"D+1",col_14d_min:"мин. 14д",
coverage:"Покрытие маршрутов (cached API)",col_branch:"Ветка",col_target:"Целевая дата",col_cheapest:"Ближайшая дата",col_transfers:"Пересадки",col_cov_days:"Дней с ценой",col_selected:"Выбранная дата",col_estimated:"Оценочная дата"},
en:{title:"Route",generated:"Generated",summary:"Summary Ranking",details:"Route details",
status:"Status",reasons:"Reasons",breakdown:"Score breakdown",unavailable:"Rejected / unavailable",
recommendation:"Recommendation",top_pick:"Top pick",no_ok:"No available routes",
col_rank:"#",col_route:"Route",col_status:"Status",col_price:"Price",col_airlines:"Airlines",col_duration:"Duration",col_score:"Score",col_visa:"Visa risk",col_operational_disruption:"Operational disruption risk",
visa_required:"Transit visa required",visa_may_required:"Transit visa may be required",
visa_check:"Check transit rules before buying",visa_risk_label:"Visa risk",price_compare:"Price comparison",available_airlines:"Available airlines",
ops_disruption_label:"Operational disruption risk",ops_elevated:"Possible airline advisories to avoid the region",ops_high:"Possible airspace closure, airport closure, or mass cancellations",ops_check:"Check current security situation before buying",ops_disclaimer:"Passenger logistics disruption risk (cancellations, delays), not geographic proximity to conflict",
note_cached:"Cached price data",note_baggage:"Baggage unknown",note_schedule:"Schedule details may be incomplete",bag_unknown:"Baggage unknown",
note_browser:"Browser-collected data",note_browser_source:"Collected from visible Aviasales search page",note_browser_unstable:"May be incomplete or unstable",
note_connection_verified:"Connection verified",note_connection_unverified:"Connection timing not fully verified",
note_no_price_target:"No price on target date",note_price_nearby:"Price from nearby date (cached)",note_route_incomplete:"Incomplete route data",
status_partial:"partial",
col_leg:"Leg",col_airline:"Airline",col_available_airlines:"Available airlines",col_flight:"Flight",col_date:"Date",col_leg_price:"Price",
col_price_d1:"D-1",col_price_d:"D",col_price_d1p:"D+1",col_14d_min:"14d min",
coverage:"Route coverage (cached API)",col_branch:"Branch",col_target:"Target date",col_cheapest:"Cheapest near target",col_transfers:"Transfers",col_cov_days:"Coverage days",col_selected:"Selected date",col_estimated:"Estimated date"}};
function setLang(loc){
localStorage.setItem("report-lang",loc);
document.querySelectorAll("[data-i18n]").forEach(el=>{
const k=el.getAttribute("data-i18n");if(i18n[loc][k])el.textContent=i18n[loc][k];});
document.getElementById("btn-ru").classList.toggle("active",loc==="ru");
document.getElementById("btn-en").classList.toggle("active",loc==="en");
document.querySelectorAll("[data-reason]").forEach(el=>{
const code=el.getAttribute("data-reason");
const labels=` + reasonLabelsJS() + `;
if(labels[code]&&labels[code][loc])el.textContent=labels[code][loc];});}
document.addEventListener("DOMContentLoaded",()=>setLang(localStorage.getItem("report-lang")||"ru"));
</script>`)

	b.WriteString("</body></html>")
	return b.String()
}

func reasonLabelsJS() string {
	m := map[string][2]string{}
	for _, c := range model.AllReasonCodes() {
		m[string(c)] = [2]string{model.ReasonLabel(c, "ru"), model.ReasonLabel(c, "en")}
	}
	b, _ := json.Marshal(m)
	return string(b)
}

func formatTripTitle(trip model.TripMeta) string {
	outbound := trip.DepartureDate
	if trip.OutboundForwardDays > 0 {
		end, _ := addDaysString(trip.DepartureDate, trip.OutboundForwardDays)
		outbound = fmt.Sprintf("%s…%s", trip.DepartureDate, end)
	}
	title := fmt.Sprintf("%s ⇄ %s — %s", trip.Origin, trip.Destination, outbound)
	if trip.ReturnDate != "" {
		ret := trip.ReturnDate
		if trip.ReturnDateEnd != "" && trip.ReturnDateEnd != trip.ReturnDate {
			ret = fmt.Sprintf("%s…%s", trip.ReturnDate, trip.ReturnDateEnd)
		}
		title += fmt.Sprintf(" / %s", ret)
	}
	return title
}

func addDaysString(date string, days int) (string, error) {
	t, err := time.Parse("2006-01-02", date)
	if err != nil {
		return date, err
	}
	return t.AddDate(0, 0, days).Format("2006-01-02"), nil
}

func formatBranchPrice(br model.BranchResult, locale string) string {
	if br.Offer != nil && br.Offer.Price.Amount == 0 {
		if locale == "ru" {
			return "нет цены"
		}
		return "no price"
	}
	var base string
	if br.PriceComparison != nil && br.PriceComparison.PriceTarget != nil {
		base = br.PriceComparison.FormatPriceComparisonCompact(locale)
	} else if br.Offer != nil && br.Offer.Price.Amount > 0 {
		base = fmt.Sprintf("%s %s", formatMoney(br.Offer.Price), br.Offer.Price.Currency)
	} else if br.PriceComparison != nil && br.PriceComparison.PriceWindowMin != nil {
		if locale == "ru" {
			base = fmt.Sprintf("мин. %dд %s", br.PriceComparison.PriceWindowDays, formatOptMoney(br.PriceComparison.PriceWindowMin))
		} else {
			base = fmt.Sprintf("%dd min %s", br.PriceComparison.PriceWindowDays, formatOptMoney(br.PriceComparison.PriceWindowMin))
		}
		if br.PriceComparison.PriceWindowMinDate != "" {
			base += " (" + br.PriceComparison.PriceWindowMinDate + ")"
		}
	} else {
		if locale == "ru" {
			return "нет цены"
		}
		return "no price"
	}
	if br.Status == model.StatusPartial {
		tag := "partial"
		if locale == "ru" {
			tag = "частично"
		}
		return base + " [" + tag + "]"
	}
	return base
}

func renderPartialStatusNotes(br model.BranchResult) string {
	if br.Status != model.StatusPartial {
		return ""
	}
	var b strings.Builder
	b.WriteString(`<ul class="data-notes">`)
	for _, code := range br.ReasonCodes {
		b.WriteString(fmt.Sprintf(`<li data-reason="%s"></li>`, code))
	}
	if br.Offer != nil {
		for _, note := range br.Offer.Notes {
			switch note {
			case model.NoteNoPriceOnTarget:
				b.WriteString(`<li data-i18n="note_no_price_target"></li>`)
			case model.NotePriceFromNearbyDate:
				b.WriteString(`<li data-i18n="note_price_nearby"></li>`)
			case model.NoteRouteDataIncomplete:
				b.WriteString(`<li data-i18n="note_route_incomplete"></li>`)
			case model.NoteEstimatedDate:
				b.WriteString(`<li data-i18n="note_price_nearby"></li>`)
			}
		}
	}
	b.WriteString(`</ul>`)
	return b.String()
}

func renderBranchCoverageForBranch(rows []model.RouteCoverageRow, branchID string) string {
	var filtered []model.RouteCoverageRow
	for _, r := range rows {
		if r.BranchID == branchID {
			filtered = append(filtered, r)
		}
	}
	if len(filtered) == 0 {
		return ""
	}
	return renderCoverageTable(filtered)
}

func formatBranchAirlines(br model.BranchResult) string {
	if len(br.LegAirlines) == 0 {
		if br.Offer == nil {
			return "n/a"
		}
		var parts []string
		for _, leg := range br.Offer.LegDetails {
			parts = append(parts, fmt.Sprintf("%s→%s: %s", leg.From, leg.To, model.FormatAirlineList(leg.AvailableAirlines)))
		}
		if len(parts) == 0 {
			return "n/a"
		}
		return strings.Join(parts, " | ")
	}
	var parts []string
	for _, la := range br.LegAirlines {
		parts = append(parts, fmt.Sprintf("%s→%s: %s", la.From, la.To, model.FormatAirlineList(la.AvailableAirlines)))
	}
	return strings.Join(parts, " | ")
}

func renderBranchAirlines(br model.BranchResult) string {
	s := formatBranchAirlines(br)
	if s == "n/a" {
		return ""
	}
	return fmt.Sprintf(`<p><span data-i18n="available_airlines">Доступные авиакомпании</span>: %s</p>`, html.EscapeString(s))
}

func renderOperationalDisruptionWarning(br model.BranchResult) string {
	r := br.OperationalDisruptionRisk
	if !r.ShowBadge() {
		return ""
	}
	cls := "badge badge-med"
	key := "ops_elevated"
	if r.ShowBoldWarning() {
		cls = "regional-warn-bold"
		key = "ops_high"
	}
	penalty := ""
	if br.OperationalDisruptionPenalty > 0 {
		penalty = fmt.Sprintf(" (-%.0f)", br.OperationalDisruptionPenalty)
	}
	return fmt.Sprintf(`<p class="%s"><strong data-i18n="%s"></strong>%s — <span data-i18n="ops_check"></span> (<span data-i18n="ops_disruption_label"></span>: %s)</p><p class="risk-note"><em data-i18n="ops_disclaimer"></em></p>`,
		cls, key, penalty, html.EscapeString(string(r)))
}

func renderRiskBadges(o *model.Offer) string {
 cls := "badge-low"
 if o.VisaRisk == model.RiskMedium { cls = "badge-med" }
 if o.VisaRisk == model.RiskHigh || o.VisaRisk == model.RiskRejected { cls = "badge-high" }
 s := fmt.Sprintf("<span class=\"badge %s\">visa:%s</span>", cls, o.VisaRisk)
 if o.SelfTransfer { s += `<span class="badge badge-med">self-transfer</span>` }
 if o.CheckedBaggageKg != nil {
   s += fmt.Sprintf(`<span class="badge badge-low">bag:%dkg</span>`, *o.CheckedBaggageKg)
 } else if o.DataQuality.AllowsUnknownBaggage(true) {
   s += `<span class="badge badge-med" data-i18n="bag_unknown">Baggage unknown</span>`
 }
 return "<p>" + s + "</p>"
}

func renderVisaWarning(cat model.VisaCategory) string {
	if !cat.ShowBoldWarning() && !cat.ShowBadge() {
		return ""
	}
	key := "visa_may_required"
	cls := "badge badge-med"
	if cat.ShowBoldWarning() {
		key = "visa_required"
		cls = "visa-warn-bold"
	}
	return fmt.Sprintf(`<p class="%s"><strong data-i18n="%s"></strong> — <span data-i18n="visa_check"></span> (<span data-i18n="visa_risk_label"></span>: %s)</p>`,
		cls, key, html.EscapeString(string(cat)))
}

func renderCoverageTable(rows []model.RouteCoverageRow) string {
	if len(rows) == 0 {
		return "<p>—</p>"
	}
	var b strings.Builder
	b.WriteString(`<table class="coverage-table"><thead><tr>`)
	for _, k := range []string{"col_branch", "col_leg", "col_target", "col_price_d1", "col_price_d", "col_price_d1p", "col_14d_min", "col_airline", "col_available_airlines", "col_cov_days", "col_selected", "col_estimated"} {
		b.WriteString(fmt.Sprintf(`<th data-i18n="%s">%s</th>`, k, k))
	}
	b.WriteString(`</tr></thead><tbody>`)
	for _, r := range rows {
		priceD := formatOptMoney(r.PriceTarget)
		priceDm1 := formatOptMoney(r.PriceMinus1)
		priceDp1 := formatOptMoney(r.PricePlus1)
		winMin := formatOptMoney(r.PriceWindowMin)
		if r.PriceWindowMinDate != "" && winMin != "" {
			winMin += " (" + r.PriceWindowMinDate + ")"
		}
		estimated := "no"
		if r.EstimatedDate {
			estimated = "yes"
		}
		b.WriteString("<tr>")
		b.WriteString(fmt.Sprintf("<td>%s</td>", html.EscapeString(r.BranchName)))
		b.WriteString(fmt.Sprintf("<td>%s→%s</td>", html.EscapeString(r.From), html.EscapeString(r.To)))
		b.WriteString(fmt.Sprintf("<td>%s</td>", html.EscapeString(r.TargetDate)))
		b.WriteString(fmt.Sprintf("<td>%s</td>", priceDm1))
		b.WriteString(fmt.Sprintf("<td>%s</td>", priceD))
		b.WriteString(fmt.Sprintf("<td>%s</td>", priceDp1))
		b.WriteString(fmt.Sprintf("<td>%s</td>", winMin))
		b.WriteString(fmt.Sprintf("<td>%s</td>", html.EscapeString(r.Airline)))
		b.WriteString(fmt.Sprintf("<td>%s</td>", html.EscapeString(model.FormatAirlineList(r.AvailableAirlines))))
		b.WriteString(fmt.Sprintf("<td>%d</td>", r.CoverageDays))
		b.WriteString(fmt.Sprintf("<td>%s</td>", html.EscapeString(r.SelectedDate)))
		b.WriteString(fmt.Sprintf("<td>%s</td>", estimated))
		b.WriteString("</tr>")
	}
	b.WriteString(`</tbody></table>`)
	return b.String()
}

func renderConnectionStatus(o *model.Offer) string {
	key := "note_connection_verified"
	if !o.ConnectionVerified {
		key = "note_connection_unverified"
	}
	return fmt.Sprintf(`<p data-i18n="%s"></p>`, key)
}

func renderLegTable(o *model.Offer) string {
	if len(o.LegDetails) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString(`<table class="leg-table"><thead><tr>`)
	for _, k := range []string{"col_leg", "col_airline", "col_available_airlines", "col_flight", "col_date", "col_leg_price"} {
		b.WriteString(fmt.Sprintf(`<th data-i18n="%s">%s</th>`, k, k))
	}
	b.WriteString(`</tr></thead><tbody>`)
	for _, leg := range o.LegDetails {
		date := leg.SearchDate
		if date == "" && !leg.Departure.IsZero() {
			date = leg.Departure.Format("2006-01-02")
		}
		b.WriteString("<tr>")
		b.WriteString(fmt.Sprintf("<td>%s→%s</td>", html.EscapeString(leg.From), html.EscapeString(leg.To)))
		b.WriteString(fmt.Sprintf("<td>%s</td>", html.EscapeString(leg.Airline)))
		b.WriteString(fmt.Sprintf("<td>%s</td>", html.EscapeString(model.FormatAirlineList(leg.AvailableAirlines))))
		b.WriteString(fmt.Sprintf("<td>%s</td>", html.EscapeString(leg.FlightNumber)))
		b.WriteString(fmt.Sprintf("<td>%s</td>", html.EscapeString(date)))
		b.WriteString(fmt.Sprintf("<td>%s %s</td>", formatMoney(leg.Price), html.EscapeString(leg.Price.Currency)))
		b.WriteString("</tr>")
	}
	b.WriteString(`</tbody></table>`)
	return b.String()
}

func renderDataQualityNotes(o *model.Offer) string {
 if o.DataQuality != model.DataQualityCached && o.DataQuality != model.DataQualityBrowserCollected {
  return ""
 }
 var b strings.Builder
 b.WriteString(`<ul class="data-notes">`)
 if o.DataQuality == model.DataQualityBrowserCollected {
  for _, key := range []string{"note_browser", "note_browser_source", "note_browser_unstable", "note_baggage"} {
   b.WriteString(fmt.Sprintf(`<li data-i18n="%s"></li>`, key))
  }
 } else {
  for _, key := range []string{"note_cached", "note_baggage", "note_schedule"} {
   b.WriteString(fmt.Sprintf(`<li data-i18n="%s"></li>`, key))
  }
 }
 for _, note := range o.Notes {
  switch note {
  case model.NoteNoPriceOnTarget:
   b.WriteString(`<li data-i18n="note_no_price_target"></li>`)
  case model.NotePriceFromNearbyDate, model.NoteEstimatedDate:
   b.WriteString(`<li data-i18n="note_price_nearby"></li>`)
  case model.NoteRouteDataIncomplete:
   b.WriteString(`<li data-i18n="note_route_incomplete"></li>`)
  }
 }
 b.WriteString(`</ul>`)
 return b.String()
}

func formatOptMoney(m *model.Money) string {
	if m == nil {
		return "n/a"
	}
	return fmt.Sprintf("%s %s", formatMoney(*m), m.Currency)
}

func topBranch(r *model.EvaluationResult) *model.BranchResult {
 for i := range r.Branches {
  if r.Branches[i].Status == model.StatusOK {
   return &r.Branches[i]
  }
 }
 return nil
}

func formatMoney(m model.Money) string {
 return fmt.Sprintf("%.2f", float64(m.Amount)/100)
}

func formatDuration(d time.Duration) string {
 h := int(d.Hours())
 m := int(d.Minutes()) % 60
 return fmt.Sprintf("%dh %dm", h, m)
}

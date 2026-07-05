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
details{margin:.75rem 0;border:1px solid #ddd;border-radius:6px;padding:.5rem 1rem;background:#fff}
.recommendation{background:#e8f4fd;padding:1rem;border-radius:8px;margin-top:1.5rem}`)
	b.WriteString("</style></head><body>")

	b.WriteString("<header><div>")
	b.WriteString(fmt.Sprintf("<h1><span data-i18n=\"title\">%s</span></h1>", html.EscapeString(fmt.Sprintf("%s → %s — %s", result.Trip.Origin, result.Trip.Destination, result.Trip.DepartureDate))))
	b.WriteString(fmt.Sprintf("<p><span data-i18n=\"generated\">Сгенерировано</span>: %s</p>", html.EscapeString(ts)))
	b.WriteString("</div><div class=\"lang-switch\">")
	b.WriteString("<button id=\"btn-ru\" class=\"active\" onclick=\"setLang('ru')\">RU</button>")
	b.WriteString("<button id=\"btn-en\" onclick=\"setLang('en')\">EN</button></div></header>")

	b.WriteString("<h2 data-i18n=\"summary\">Сводный рейтинг</h2>")
	b.WriteString("<table id=\"summary-table\"><thead><tr>")
	for _, k := range []string{"rank", "route", "status", "price", "duration", "score"} {
		b.WriteString(fmt.Sprintf("<th data-i18n=\"col_%s\">%s</th>", k, k))
	}
	b.WriteString("</tr></thead><tbody>")
	rank := 1
	for _, br := range result.Branches {
		if br.Status != model.StatusOK {
			continue
		}
		b.WriteString("<tr>")
		b.WriteString(fmt.Sprintf("<td>%d</td><td>%s</td><td>%s</td>", rank, html.EscapeString(br.BranchName), br.Status))
		if br.Offer != nil {
			b.WriteString(fmt.Sprintf("<td>%s %s</td><td>%s</td>", formatMoney(br.Offer.Price), html.EscapeString(br.Offer.Price.Currency), formatDuration(br.Offer.TotalDuration)))
		} else {
			b.WriteString("<td></td><td></td>")
		}
		if br.Score != nil {
			b.WriteString(fmt.Sprintf("<td>%.1f</td>", *br.Score))
		} else {
			b.WriteString("<td></td>")
		}
		b.WriteString("</tr>")
		rank++
	}
	b.WriteString("</tbody></table>")

	b.WriteString("<h2 data-i18n=\"details\">Детали маршрутов</h2>")
	for _, br := range result.Branches {
		b.WriteString("<details><summary>")
		b.WriteString(html.EscapeString(br.BranchName))
		if br.Score != nil {
			b.WriteString(fmt.Sprintf(" — %.1f", *br.Score))
		}
		b.WriteString("</summary>")
		b.WriteString(fmt.Sprintf("<p><span data-i18n=\"status\">Статус</span>: %s</p>", br.Status))
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
			for _, seg := range o.Segments {
				b.WriteString(fmt.Sprintf("<p>%s %s→%s %s %s</p>",
					html.EscapeString(seg.FlightNumber), seg.From, seg.To,
					seg.Departure.Format("2006-01-02 15:04 MST"), seg.Arrival.Format("2006-01-02 15:04 MST")))
			}
		}
		b.WriteString("</details>")
	}

	b.WriteString("<h2 data-i18n=\"unavailable\">Недоступные / отклонённые</h2><ul>")
	for _, br := range result.Branches {
		if br.Status == model.StatusOK {
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
col_rank:"#",col_route:"Маршрут",col_status:"Статус",col_price:"Цена",col_duration:"Длительность",col_score:"Оценка"},
en:{title:"Route",generated:"Generated",summary:"Summary Ranking",details:"Route details",
status:"Status",reasons:"Reasons",breakdown:"Score breakdown",unavailable:"Rejected / unavailable",
recommendation:"Recommendation",top_pick:"Top pick",no_ok:"No available routes",
col_rank:"#",col_route:"Route",col_status:"Status",col_price:"Price",col_duration:"Duration",col_score:"Score"}};
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

func renderRiskBadges(o *model.Offer) string {
 cls := "badge-low"
 if o.VisaRisk == model.RiskMedium { cls = "badge-med" }
 if o.VisaRisk == model.RiskHigh || o.VisaRisk == model.RiskRejected { cls = "badge-high" }
 s := fmt.Sprintf("<span class=\"badge %s\">visa:%s</span>", cls, o.VisaRisk)
 if o.SelfTransfer { s += `<span class="badge badge-med">self-transfer</span>` }
 if o.CheckedBaggageKg != nil {
   s += fmt.Sprintf(`<span class="badge badge-low">bag:%dkg</span>`, *o.CheckedBaggageKg)
 }
 return "<p>" + s + "</p>"
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

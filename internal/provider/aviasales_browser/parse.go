package aviasales_browser

import (
	"encoding/json"
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/kneumoin/nepal/internal/model"
)

var priceTextRe = regexp.MustCompile(`(?i)(USD|RUB|EUR|\$|₽|€)\s*([\d\s]+)|([\d\s]+)\s*(USD|RUB|EUR|\$|₽|€)`)

// ExtractedOffer is raw data parsed from a visible search page.
type ExtractedOffer struct {
	PriceAmount     int64  `json:"price_amount"`
	Currency        string `json:"currency"`
	Airline         string `json:"airline,omitempty"`
	Departure       string `json:"departure,omitempty"`
	Arrival         string `json:"arrival,omitempty"`
	DepartureClock  string `json:"departure_clock,omitempty"` // HH:MM from UI when no ISO timestamp
	ArrivalClock    string `json:"arrival_clock,omitempty"`
	DurationMinutes int    `json:"duration_minutes,omitempty"`
	Stops           int      `json:"stops,omitempty"`
	SourceLabel     string   `json:"source_label,omitempty"`
	BaggageKg       *int     `json:"baggage_kg,omitempty"`
	Airlines        []string `json:"airlines,omitempty"`        // all carriers shown on the card
	BookingURL      string   `json:"booking_url,omitempty"`     // agency link if visible in DOM
	SearchPageURL   string   `json:"search_page_url,omitempty"` // fallback: the search results page
}

type CachedPage struct {
	URL         string           `json:"url"`
	CollectedAt time.Time        `json:"collected_at"`
	Extracted   []ExtractedOffer `json:"extracted"`
}

var browserNotes = []string{
	"Browser-collected data",
	"Collected from visible Aviasales search page",
	"May be incomplete or unstable",
	"Baggage unknown",
}

func detectCaptcha(html string) bool {
	lower := strings.ToLower(html)
	// If real ticket results are present, the page is not a captcha wall even
	// though the reCAPTCHA JS library is always bundled.
	if strings.Contains(lower, "ticket-preview") || strings.Contains(lower, "aviasales-browser-offer") {
		return false
	}
	for _, m := range captchaMarkers {
		if strings.Contains(lower, m) {
			return true
		}
	}
	return false
}

func parseHTML(html string, from, to, date string) ([]ExtractedOffer, error) {
	return parseHTMLWithPageURL(html, from, to, date, "")
}

func parseHTMLWithPageURL(html, from, to, date, pageURL string) ([]ExtractedOffer, error) {
	if detectCaptcha(html) {
		return nil, fmt.Errorf("captcha or bot-check detected")
	}
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return nil, err
	}
	var offers []ExtractedOffer
	doc.Find(selectorOfferCard).Each(func(_ int, sel *goquery.Selection) {
		if o, ok := extractOfferNode(sel); ok {
			if o.BookingURL == "" {
				o.BookingURL = findBookingHref(sel)
			}
			if o.SearchPageURL == "" {
				o.SearchPageURL = pageURL
			}
			offers = append(offers, o)
		}
	})
	if len(offers) == 0 {
		return nil, fmt.Errorf("no visible offers found")
	}
	return offers, nil
}

// findBookingHref looks for a purchase/book link inside an offer card.
func findBookingHref(sel *goquery.Selection) string {
	var href string
	sel.Find("a[href]").EachWithBreak(func(_ int, a *goquery.Selection) bool {
		link, ok := a.Attr("href")
		if !ok || link == "" || strings.HasPrefix(link, "#") {
			return true
		}
		lower := strings.ToLower(link + " " + strings.TrimSpace(a.Text()))
		if strings.Contains(lower, "javascript:") {
			return true
		}
		// Prefer explicit buy/book actions; otherwise keep first external http(s) link.
		if strings.Contains(lower, "купить") || strings.Contains(lower, "buy") ||
			strings.Contains(lower, "book") || strings.Contains(lower, "билет") {
			href = link
			return false
		}
		if href == "" && (strings.HasPrefix(link, "http://") || strings.HasPrefix(link, "https://")) {
			href = link
		}
		return true
	})
	return href
}

// clockTimeInTagRe matches HH:MM values that sit between HTML tags (>16:05<),
// which keeps departure/arrival times separated on real Aviasales cards.
var clockTimeInTagRe = regexp.MustCompile(`>\s*(\d{1,2}:\d{2})\s*<`)

func extractOfferNode(sel *goquery.Selection) (ExtractedOffer, bool) {
	priceStr, _ := sel.Attr(attrPrice)
	cur, _ := sel.Attr(attrCurrency)

	// Real Aviasales: price lives in a child [data-test-id="price"] node.
	if priceStr == "" {
		if node := sel.Find(selectorPriceNode).First(); node.Length() > 0 {
			amount, c := parsePriceText(strings.TrimSpace(node.Text()))
			if amount > 0 {
				priceStr = fmt.Sprintf("%.0f", amount)
				if cur == "" {
					cur = c
				}
			}
		}
	}
	// Fallback: parse whole card text.
	if priceStr == "" {
		amount, c := parsePriceText(strings.TrimSpace(sel.Text()))
		if amount <= 0 {
			return ExtractedOffer{}, false
		}
		priceStr = fmt.Sprintf("%.0f", amount)
		if cur == "" {
			cur = c
		}
	}
	price, err := strconv.ParseFloat(strings.ReplaceAll(priceStr, " ", ""), 64)
	if err != nil || price <= 0 {
		return ExtractedOffer{}, false
	}
	if cur == "" {
		cur = "USD"
	}

	airline, _ := sel.Attr(attrAirline)
	airlines := airlinesFromImg(sel)
	if airline == "" && len(airlines) > 0 {
		airline = airlines[0]
	}
	dep, _ := sel.Attr(attrDep)
	arr, _ := sel.Attr(attrArr)
	durStr, _ := sel.Attr(attrDuration)
	stopsStr, _ := sel.Attr(attrStops)
	source, _ := sel.Attr(attrSource)
	bagStr, _ := sel.Attr(attrBaggage)

	o := ExtractedOffer{
		PriceAmount: int64(math.Round(price * 100)),
		Currency:    strings.ToUpper(cur),
		Airline:     strings.TrimSpace(airline),
		Airlines:    airlines,
		Departure:   dep,
		Arrival:     arr,
		SourceLabel: source,
	}
	// Real page: departure/arrival are HH:MM text nodes, not ISO attrs.
	if o.Departure == "" || o.Arrival == "" {
		if d, a, ok := clockTimesFromCard(sel); ok {
			if o.Departure == "" {
				o.DepartureClock = d
			}
			if o.Arrival == "" {
				o.ArrivalClock = a
			}
		}
	}
	// Real page: stops derived from segment connectors.
	if stopsStr == "" {
		if n := sel.Find(selectorConnector).Length(); n > 0 {
			o.Stops = n - 1
			if o.Stops < 0 {
				o.Stops = 0
			}
		}
	}
	if durStr != "" {
		if d, err := strconv.Atoi(durStr); err == nil {
			o.DurationMinutes = d
		}
	}
	if stopsStr != "" {
		if s, err := strconv.Atoi(stopsStr); err == nil {
			o.Stops = s
		}
	}
	if bagStr != "" {
		if b, err := strconv.Atoi(bagStr); err == nil {
			o.BaggageKg = &b
		}
	}
	return o, true
}

// airlinesFromImg reads airline names from <img alt="..."> nodes in the card
// (Aviasales shows one logo per operating carrier), de-duplicated in order.
func airlinesFromImg(sel *goquery.Selection) []string {
	var out []string
	seen := map[string]bool{}
	sel.Find("img[alt]").Each(func(_ int, img *goquery.Selection) {
		alt := strings.TrimSpace(img.AttrOr("alt", ""))
		if len(alt) < 3 || seen[alt] {
			return
		}
		seen[alt] = true
		out = append(out, alt)
	})
	return out
}

// clockTimesFromCard extracts the first (departure) and last (arrival) HH:MM
// times shown on the card, reading inner HTML so times in separate nodes stay split.
func clockTimesFromCard(sel *goquery.Selection) (dep, arr string, ok bool) {
	h, err := sel.Html()
	if err != nil || h == "" {
		h = sel.Text()
	}
	matches := clockTimeInTagRe.FindAllStringSubmatch(h, -1)
	if len(matches) == 0 {
		return "", "", false
	}
	dep = matches[0][1]
	arr = matches[len(matches)-1][1]
	return dep, arr, true
}

// unicodeSpaces are separators Aviasales uses inside prices (narrow/no-break spaces).
var unicodeSpaceReplacer = strings.NewReplacer(
	"\u202f", " ", // narrow no-break space
	"\u00a0", " ", // no-break space
	"\u2009", " ", // thin space
	"\u2007", " ", // figure space
)

func parsePriceText(s string) (float64, string) {
	s = unicodeSpaceReplacer.Replace(s)
	m := priceTextRe.FindStringSubmatch(s)
	if m == nil {
		return 0, ""
	}
	cur := "USD"
	var num string
	if m[1] != "" {
		cur = normalizeCurrency(m[1])
		num = m[2]
	} else {
		num = m[3]
		cur = normalizeCurrency(m[4])
	}
	num = strings.ReplaceAll(strings.TrimSpace(num), " ", "")
	v, err := strconv.ParseFloat(num, 64)
	if err != nil {
		return 0, ""
	}
	return v, cur
}

func parseClock(s string) (hour, min int, ok bool) {
	parts := strings.SplitN(strings.TrimSpace(s), ":", 2)
	if len(parts) != 2 {
		return 0, 0, false
	}
	h, err1 := strconv.Atoi(parts[0])
	m, err2 := strconv.Atoi(parts[1])
	if err1 != nil || err2 != nil || h < 0 || h > 23 || m < 0 || m > 59 {
		return 0, 0, false
	}
	return h, m, true
}

func normalizeCurrency(s string) string {
	s = strings.ToUpper(strings.TrimSpace(s))
	switch s {
	case "$":
		return "USD"
	case "₽":
		return "RUB"
	case "€":
		return "EUR"
	default:
		return s
	}
}

func pickBestOffer(extracted []ExtractedOffer) *ExtractedOffer {
	if len(extracted) == 0 {
		return nil
	}
	best := extracted[0]
	for _, o := range extracted[1:] {
		if o.PriceAmount < best.PriceAmount {
			best = o
		}
	}
	return &best
}

func mapToOffer(ex ExtractedOffer, from, to, date string) model.Offer {
	seg := buildSegment(from, to, date, ex)
	total := time.Duration(ex.DurationMinutes) * time.Minute
	if total == 0 && seg.Duration > 0 {
		total = seg.Duration
	}
	notes := append([]string(nil), browserNotes...)
	if ex.BaggageKg == nil {
		// keep baggage unknown note
	} else {
		notes = append(notes, fmt.Sprintf("Baggage visible: %dkg", *ex.BaggageKg))
	}
	if ex.BookingURL != "" {
		notes = append(notes, "Booking link: "+ex.BookingURL)
	} else if ex.SearchPageURL != "" {
		notes = append(notes, "Search page: "+ex.SearchPageURL)
	}
	if len(ex.Airlines) > 1 {
		notes = append(notes, "Carriers: "+strings.Join(ex.Airlines, ", "))
	}
	return model.Offer{
		Provider:          "aviasales_browser",
		Segments:          []model.Segment{seg},
		Price:             model.Money{Amount: ex.PriceAmount, Currency: ex.Currency},
		TotalDuration:     total,
		CheckedBaggageKg:  ex.BaggageKg,
		VisaRisk:          model.RiskLow,
		DataQuality:       model.DataQualityBrowserCollected,
		AvailableAirlines: ex.Airlines,
		Notes:             notes,
	}
}

func buildSegment(from, to, date string, ex ExtractedOffer) model.Segment {
	fromLoc := airportLoc(from)
	toLoc := airportLoc(to)
	day, _ := time.Parse("2006-01-02", date)

	var dep, arr time.Time
	if ex.Departure != "" {
		if t, err := time.Parse(time.RFC3339, ex.Departure); err == nil {
			dep = t.In(fromLoc)
		}
	}
	if ex.Arrival != "" {
		if t, err := time.Parse(time.RFC3339, ex.Arrival); err == nil {
			arr = t.In(toLoc)
		}
	}
	// UI-provided HH:MM clock times (real Aviasales page).
	if dep.IsZero() && ex.DepartureClock != "" {
		if h, m, ok := parseClock(ex.DepartureClock); ok {
			dep = time.Date(day.Year(), day.Month(), day.Day(), h, m, 0, 0, fromLoc)
		}
	}
	if arr.IsZero() && ex.ArrivalClock != "" {
		if h, m, ok := parseClock(ex.ArrivalClock); ok {
			arr = time.Date(day.Year(), day.Month(), day.Day(), h, m, 0, 0, toLoc)
		}
	}
	dur := time.Duration(ex.DurationMinutes) * time.Minute
	if dep.IsZero() {
		dep = time.Date(day.Year(), day.Month(), day.Day(), 8, 0, 0, 0, fromLoc)
	}
	if arr.IsZero() {
		if dur > 0 {
			arr = dep.Add(dur).In(toLoc)
		} else {
			arr = dep.Add(5 * time.Hour).In(toLoc)
			dur = 5 * time.Hour
		}
	} else if dur == 0 {
		dur = arr.Sub(dep)
	}
	fn := ""
	if ex.Airline != "" {
		fn = ex.Airline + "000"
	}
	return model.Segment{
		From: from, To: to,
		Departure: dep, Arrival: arr,
		Airline: ex.Airline, FlightNumber: fn,
		Duration: dur,
	}
}

func airportLoc(iata string) *time.Location {
	tz := map[string]string{
		"MOW": "Europe/Moscow", "DOH": "Asia/Qatar", "DXB": "Asia/Dubai",
		"DEL": "Asia/Kolkata", "IST": "Europe/Istanbul", "TFU": "Asia/Shanghai",
		"KTM": "Asia/Kathmandu",
	}
	if name, ok := tz[iata]; ok {
		if loc, err := time.LoadLocation(name); err == nil {
			return loc
		}
	}
	return time.UTC
}

func decodeCachedPage(raw []byte) (*CachedPage, error) {
	var p CachedPage
	if err := json.Unmarshal(raw, &p); err != nil {
		return nil, err
	}
	return &p, nil
}

// ParseHTMLForTest exposes HTML parsing for unit tests.
func ParseHTMLForTest(html, from, to, date string) ([]ExtractedOffer, error) {
	return parseHTML(html, from, to, date)
}

func encodeCachedPage(p CachedPage) ([]byte, error) {
	return json.Marshal(p)
}

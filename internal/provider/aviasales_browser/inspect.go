package aviasales_browser

import (
	"fmt"
	"sort"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

// DOMHint helps discover real Aviasales selectors from saved HTML.
type DOMHint struct {
	Title           string
	OfferLikeCount  int
	LinkCount       int
	SampleLinks     []string
	SampleClasses   []string
	CaptchaDetected bool
}

// InspectHTML analyzes a search results page for selector discovery (UI driver dev tool).
func InspectHTML(html string) DOMHint {
	h := DOMHint{Title: "Aviasales search page"}
	if detectCaptcha(html) {
		h.CaptchaDetected = true
	}
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		h.Title = "parse error: " + err.Error()
		return h
	}
	if t := strings.TrimSpace(doc.Find("title").First().Text()); t != "" {
		h.Title = t
	}

	h.OfferLikeCount = doc.Find(selectorOfferCard).Length()
	if h.OfferLikeCount == 0 {
		h.OfferLikeCount = doc.Find("[class*='ticket'], [class*='Ticket'], [data-testid*='ticket']").Length()
	}

	classFreq := map[string]int{}
	doc.Find("a[href]").Each(func(_ int, a *goquery.Selection) {
		href, _ := a.Attr("href")
		if href == "" || strings.HasPrefix(href, "#") || strings.HasPrefix(href, "javascript:") {
			return
		}
		text := strings.TrimSpace(a.Text())
		label := truncateInspect(text, 40)
		if label == "" {
			label = "(no text)"
		}
		entry := truncateInspect(href, 120) + " :: " + label
		if len(h.SampleLinks) < 15 {
			h.SampleLinks = append(h.SampleLinks, entry)
		}
		h.LinkCount++
	})

	doc.Find("[class]").Each(func(_ int, s *goquery.Selection) {
		cls, _ := s.Attr("class")
		for _, c := range strings.Fields(cls) {
			lower := strings.ToLower(c)
			if strings.Contains(lower, "ticket") || strings.Contains(lower, "product") ||
				strings.Contains(lower, "offer") || strings.Contains(lower, "flight") {
				classFreq[c]++
			}
		}
	})
	type kv struct{ k string; v int }
	var ranked []kv
	for k, v := range classFreq {
		ranked = append(ranked, kv{k, v})
	}
	sort.Slice(ranked, func(i, j int) bool { return ranked[i].v > ranked[j].v })
	for i, item := range ranked {
		if i >= 12 {
			break
		}
		h.SampleClasses = append(h.SampleClasses, fmt.Sprintf("%s (%d)", item.k, item.v))
	}
	return h
}

func truncateInspect(s string, n int) string {
	s = strings.Join(strings.Fields(s), " ")
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

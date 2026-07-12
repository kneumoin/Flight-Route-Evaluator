package links

import (
	"fmt"
	"net/url"
	"strings"
	"time"
)

const searchBaseURL = "https://www.aviasales.ru/search/"

// OneWaySearchURL builds a public Aviasales one-way search URL.
// Format: https://www.aviasales.ru/search/MOW2809KTM1
func OneWaySearchURL(origin, dest, date string, passengers int) (string, error) {
	origin = strings.ToUpper(strings.TrimSpace(origin))
	dest = strings.ToUpper(strings.TrimSpace(dest))
	if len(origin) != 3 || len(dest) != 3 {
		return "", fmt.Errorf("invalid IATA codes: %s-%s", origin, dest)
	}
	t, err := time.Parse("2006-01-02", date)
	if err != nil {
		return "", fmt.Errorf("invalid date %q: %w", date, err)
	}
	if passengers < 1 {
		passengers = 1
	}
	code := fmt.Sprintf("%s%s%s%d", origin, t.Format("0201"), dest, passengers)
	return searchBaseURL + code, nil
}

// RoundTripShortURL builds a public Aviasales round-trip search URL.
// Format: https://www.aviasales.ru/search/MOW2609KTM13111 (out DDMM, back DDMM, passengers)
func RoundTripShortURL(origin, dest, departDate, returnDate string, passengers int) (string, error) {
	origin = strings.ToUpper(strings.TrimSpace(origin))
	dest = strings.ToUpper(strings.TrimSpace(dest))
	dep, err := time.Parse("2006-01-02", departDate)
	if err != nil {
		return "", fmt.Errorf("invalid depart date %q: %w", departDate, err)
	}
	ret, err := time.Parse("2006-01-02", returnDate)
	if err != nil {
		return "", fmt.Errorf("invalid return date %q: %w", returnDate, err)
	}
	if passengers < 1 {
		passengers = 1
	}
	code := fmt.Sprintf("%s%s%s%s%d", origin, dep.Format("0201"), dest, ret.Format("0201"), passengers)
	return searchBaseURL + code, nil
}

// RoundTripQueryURL builds an Aviasales round-trip URL with explicit query parameters.
func RoundTripQueryURL(origin, dest, departDate, returnDate string, passengers int) string {
	if passengers < 1 {
		passengers = 1
	}
	q := url.Values{}
	q.Set("origin_iata", strings.ToUpper(origin))
	q.Set("destination_iata", strings.ToUpper(dest))
	q.Set("depart_date", departDate)
	q.Set("return_date", returnDate)
	q.Set("passengers", fmt.Sprintf("%d", passengers))
	return "https://www.aviasales.ru/?" + q.Encode()
}

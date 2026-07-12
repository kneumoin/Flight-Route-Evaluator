package aviasales_browser

import "github.com/kneumoin/nepal/internal/links"

// SearchURL builds an Aviasales public search URL for a one-way leg.
func SearchURL(origin, dest, date string, passengers int) (string, error) {
	return links.OneWaySearchURL(origin, dest, date, passengers)
}

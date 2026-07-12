package aviasales_search

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"strings"
)

// BuildSignature computes the MD5 signature for the new Flight Search API (2025+).
// Algorithm: https://support.travelpayouts.com/hc/en-us/articles/210996008
//
// 1. Sort top-level JSON keys alphabetically; flatten nested objects separately.
// 2. Join all parameter values with ":" (no keys).
// 3. Prepend API token + ":".
// 4. MD5 hex digest (lowercase).
func BuildSignature(token string, req StartRequest) string {
	parts := signatureParts(req)
	base := token + ":" + strings.Join(parts, ":")
	sum := md5.Sum([]byte(base))
	return hex.EncodeToString(sum[:])
}

func signatureParts(req StartRequest) []string {
	var out []string
	out = append(out, req.CurrencyCode)
	out = append(out, req.Locale)
	out = append(out, req.Marker)
	out = append(out, req.MarketCode)
	out = append(out, searchParamsParts(req.SearchParams)...)
	return out
}

func searchParamsParts(sp SearchParams) []string {
	var out []string
	for _, d := range sp.Directions {
		out = append(out, directionParts(d)...)
	}
	out = append(out,
		fmt.Sprintf("%d", sp.Passengers.Adults),
		fmt.Sprintf("%d", sp.Passengers.Children),
		fmt.Sprintf("%d", sp.Passengers.Infants),
		sp.TripClass,
	)
	return out
}

// directionParts: keys sorted date, destination, origin.
func directionParts(d Direction) []string {
	return []string{d.Date, d.Destination, d.Origin}
}

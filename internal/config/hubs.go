package config

import (
	"fmt"
	"strings"
)

// ApprovedTransitHubs is the fixed list of one-stop hubs to Kathmandu.
var ApprovedTransitHubs = []string{
	"IST", "KWI", "DMM", "RUH", "DXB", "SHJ", "AUH", "DOH",
	"DEL", "BOM", "BLR", "CMB", "CCU", "DAC",
	"LXA", "KMG", "TFU", "CAN", "SZX", "HKG",
	"BKK", "DMK", "KUL", "SIN", "ICN", "NRT",
}

// MultiStopFirstLegHubs are extra intermediate airports allowed as the first stop on two-transfer routes.
var MultiStopFirstLegHubs = []string{"TAS", "DYU"}

var approvedHubSet map[string]bool
var multiStopFirstLegSet map[string]bool

func init() {
	approvedHubSet = make(map[string]bool, len(ApprovedTransitHubs))
	for _, h := range ApprovedTransitHubs {
		approvedHubSet[h] = true
	}
	multiStopFirstLegSet = make(map[string]bool, len(MultiStopFirstLegHubs))
	for _, h := range MultiStopFirstLegHubs {
		multiStopFirstLegSet[h] = true
	}
}

// IsApprovedHub reports whether code is on the approved transit hub list.
func IsApprovedHub(iata string) bool {
	return approvedHubSet[strings.ToUpper(strings.TrimSpace(iata))]
}

// Known invalid IATA typos (valid format but wrong code).
var invalidIATATypos = map[string]bool{
	"HGK": true, // should be HKG
}

func IsValidHubIATA(code string) bool {
	code = strings.ToUpper(strings.TrimSpace(code))
	if invalidIATATypos[code] {
		return false
	}
	return validIATA(code) && IsApprovedHub(code)
}

// IsValidIntermediateIATA reports whether code may appear between MOW and KTM on a branch leg.
func IsValidIntermediateIATA(code string) bool {
	code = strings.ToUpper(strings.TrimSpace(code))
	if invalidIATATypos[code] {
		return false
	}
	if !validIATA(code) {
		return false
	}
	return IsApprovedHub(code) || multiStopFirstLegSet[code]
}

func IsMultiStopFirstLegHub(code string) bool {
	return multiStopFirstLegSet[strings.ToUpper(strings.TrimSpace(code))]
}

func validateHubLeg(leg LegConfig, branchID string) error {
	from := strings.ToUpper(leg.From)
	to := strings.ToUpper(leg.To)
	if from != "MOW" && !IsValidIntermediateIATA(from) {
		return fmt.Errorf("branch %s: invalid hub origin %s", branchID, leg.From)
	}
	if to != "KTM" && !IsValidIntermediateIATA(to) {
		return fmt.Errorf("branch %s: invalid hub destination %s", branchID, leg.To)
	}
	return nil
}

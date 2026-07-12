package model

// OperationalDisruptionRisk reflects current passenger-facing disruption exposure at a hub.
// Default is LOW; elevated only when configured for known operational restrictions.
type OperationalDisruptionRisk string

const (
	OperationalDisruptionLow      OperationalDisruptionRisk = "LOW"
	OperationalDisruptionElevated OperationalDisruptionRisk = "ELEVATED"
	OperationalDisruptionHigh     OperationalDisruptionRisk = "HIGH"
	OperationalDisruptionUnknown  OperationalDisruptionRisk = "UNKNOWN"
)

func (r OperationalDisruptionRisk) ShowBoldWarning() bool {
	return r == OperationalDisruptionHigh
}

func (r OperationalDisruptionRisk) ShowBadge() bool {
	return r == OperationalDisruptionElevated || r == OperationalDisruptionHigh || r == OperationalDisruptionUnknown
}

// OperationalDisruptionNotes explains logistics disruption risk (not geographic proximity).
func OperationalDisruptionNotes(r OperationalDisruptionRisk) string {
	switch r {
	case OperationalDisruptionHigh:
		return "Active disruption risk: airspace closure, airport closure, or mass cancellations — not attack probability"
	case OperationalDisruptionElevated:
		return "Elevated risk: airline advisories to avoid the region; monitor NOTAM/EASA updates"
	case OperationalDisruptionUnknown:
		return "Operational disruption exposure unknown; verify current advisories"
	default:
		return ""
	}
}

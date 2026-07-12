package model

// VisaCategory is the transit visa risk category shown in reports.
type VisaCategory string

const (
	VisaCategoryLow          VisaCategory = "LOW"
	VisaCategoryMedium       VisaCategory = "MEDIUM"
	VisaCategoryHigh         VisaCategory = "HIGH"
	VisaCategoryRequiresVisa VisaCategory = "REQUIRES_VISA"
	VisaCategoryUnknown      VisaCategory = "UNKNOWN"
)

// VisaWarningLabel returns EN/RU label key for HTML i18n.
func VisaWarningLabel(cat VisaCategory, bold bool) (enKey, ruKey string) {
	switch cat {
	case VisaCategoryRequiresVisa:
		return "visa_required", "visa_required"
	case VisaCategoryHigh, VisaCategoryUnknown:
		return "visa_may_required", "visa_may_required"
	default:
		return "", ""
	}
}

func (cat VisaCategory) ShowBoldWarning() bool {
	return cat == VisaCategoryRequiresVisa
}

func (cat VisaCategory) ShowBadge() bool {
	return cat == VisaCategoryHigh || cat == VisaCategoryUnknown || cat == VisaCategoryRequiresVisa
}

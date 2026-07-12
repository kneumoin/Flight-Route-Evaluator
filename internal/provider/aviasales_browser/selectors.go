package aviasales_browser

// DOM selectors and captcha markers for Aviasales search pages.
// These are best-effort and may break when Aviasales changes UI.
// Fixture HTML uses class aviasales-browser-offer for stable tests.

const (
	// Real Aviasales results use data-test-id="ticket-preview"; the others keep
	// fixture/legacy compatibility.
	selectorOfferCard = "[data-test-id='ticket-preview'], .aviasales-browser-offer, [data-testid='ticket'], .product-list__ticket"

	// Child selectors within a real Aviasales ticket-preview card.
	selectorPriceNode   = "[data-test-id='price']"
	selectorConnector   = "[data-test-id='ticket-preview-segment-connector']"

	attrPrice    = "data-price"
	attrCurrency = "data-currency"
	attrAirline  = "data-airline"
	attrDep      = "data-departure"
	attrArr      = "data-arrival"
	attrDuration = "data-duration-minutes"
	attrStops    = "data-stops"
	attrSource   = "data-source"
	attrBaggage  = "data-baggage-kg"
)

var captchaMarkers = []string{
	"captcha",
	"recaptcha",
	"cf-challenge",
	"robot",
	"подтвердите",
	"не робот",
	"verify you are human",
	"access denied",
}

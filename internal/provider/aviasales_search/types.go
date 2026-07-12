package aviasales_search

// StartRequest is the body for POST .../search/affiliate/start
type StartRequest struct {
	Signature    string       `json:"signature"`
	Marker       string       `json:"marker"`
	Locale       string       `json:"locale"`
	CurrencyCode string       `json:"currency_code,omitempty"`
	MarketCode   string       `json:"market_code,omitempty"`
	SearchParams SearchParams `json:"search_params"`
}

type SearchParams struct {
	Directions []Direction `json:"directions"`
	TripClass  string      `json:"trip_class"`
	Passengers Passengers  `json:"passengers"`
}

type Direction struct {
	Origin      string `json:"origin"`
	Destination string `json:"destination"`
	Date        string `json:"date"`
}

type Passengers struct {
	Adults   int `json:"adults"`
	Children int `json:"children"`
	Infants  int `json:"infants"`
}

// StartResponse from affiliate/start.
type StartResponse struct {
	SearchID   string `json:"search_id"`
	ResultsURL string `json:"results_url"`
}

// ResultsRequest body for POST {results_url}/search/affiliate/results
type ResultsRequest struct {
	SearchID            string `json:"search_id"`
	LastUpdateTimestamp int64  `json:"last_update_timestamp"`
}

// ResultsResponse is a trimmed view of the poll response.
type ResultsResponse struct {
	LastUpdateTimestamp int64  `json:"last_update_timestamp"`
	IsOver              bool   `json:"is_over"`
	Raw                 []byte `json:"-"`
}

// ClickResponse from GET .../searches/{id}/clicks/{proposal_id}
type ClickResponse struct {
	URL            string `json:"url"`
	Method         string `json:"method"`
	ExpireAtUnixSec int64  `json:"expire_at_unix_sec"`
	StrClickID     string `json:"str_click_id"`
}

// Config holds credentials for the Search API driver.
type Config struct {
	Token     string // x-affiliate-user-id / TRAVELPAYOUTS_TOKEN
	Marker    string // partner ID, TRAVELPAYOUTS_MARKER
	RealHost  string // x-real-host, e.g. localhost or your domain
	UserIP    string // x-user-ip, end-user IP (demo: 127.0.0.1)
	Locale    string // e.g. ru
	MarketCode string // e.g. ru
	Currency  string // e.g. usd
}

package aviasales_browser

import "time"

// Options configures the experimental local-only browser provider.
type Options struct {
	Headful    bool
	RateLimit  time.Duration
	CacheOnly  bool
	Timeout    time.Duration
	Verbose    bool
	CacheDir   string
	CacheTTL   string
	CacheEnabled bool
}

func DefaultOptions() Options {
	return Options{
		Headful:      true,
		RateLimit:    time.Minute,
		CacheOnly:    false,
		Timeout:      120 * time.Second,
		CacheEnabled: true,
		CacheDir:     ".cache",
		CacheTTL:     "24h",
	}
}

package aviasales_browser

import "context"

// PageFetcher loads rendered HTML for a search URL.
type PageFetcher interface {
	Fetch(ctx context.Context, url string) (html string, err error)
}

// StaticFetcher returns fixed HTML (tests only).
type StaticFetcher struct {
	Pages map[string]string
}

func (f *StaticFetcher) Fetch(_ context.Context, url string) (string, error) {
	if html, ok := f.Pages[url]; ok {
		return html, nil
	}
	return "", ErrPageNotFound
}

// ErrPageNotFound indicates no fixture page for the URL.
var ErrPageNotFound = errPageNotFound("page not found")

type errPageNotFound string

func (e errPageNotFound) Error() string { return string(e) }

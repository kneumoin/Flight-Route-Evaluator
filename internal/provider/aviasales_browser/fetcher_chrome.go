package aviasales_browser

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/chromedp/chromedp"
)

// ChromeFetcher opens URLs in a local Chrome/Chromium instance (headful by default).
type ChromeFetcher struct {
	Headful bool
	Timeout time.Duration
	Verbose bool
}

func (c *ChromeFetcher) Fetch(ctx context.Context, url string) (string, error) {
	if c.Verbose {
		fmt.Printf("aviasales_browser: opening %s (headful=%v)\n", url, c.Headful)
	}
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", !c.Headful),
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("no-sandbox", true),
	)
	allocCtx, cancelAlloc := chromedp.NewExecAllocator(ctx, opts...)
	defer cancelAlloc()

	browserCtx, cancelBrowser := chromedp.NewContext(allocCtx)
	defer cancelBrowser()

	timeout := c.Timeout
	if timeout <= 0 {
		timeout = 120 * time.Second
	}
	ctx, cancel := context.WithTimeout(browserCtx, timeout)
	defer cancel()

	var html string
	err := chromedp.Run(ctx,
		chromedp.Navigate(url),
		chromedp.WaitReady("body", chromedp.ByQuery),
		waitForOffersOrTimeout(45*time.Second),
		chromedp.OuterHTML("html", &html, chromedp.ByQuery),
	)
	if err != nil {
		return "", fmt.Errorf("browser fetch: %w", err)
	}
	return html, nil
}

// waitForOffersOrTimeout waits until ticket cards appear or times out (page still captured).
func waitForOffersOrTimeout(maxWait time.Duration) chromedp.Action {
	return chromedp.ActionFunc(func(ctx context.Context) error {
		deadline := time.Now().Add(maxWait)
		js := `document.querySelectorAll('.aviasales-browser-offer,[data-testid="ticket"],.product-list__ticket').length`
		for time.Now().Before(deadline) {
			var n int
			if err := chromedp.Run(ctx, chromedp.Evaluate(js, &n)); err == nil && n > 0 {
				return nil
			}
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(2 * time.Second):
			}
		}
		return nil
	})
}

type rateLimiter struct {
	mu       sync.Mutex
	interval time.Duration
	last     time.Time
}

func newRateLimiter(interval time.Duration) *rateLimiter {
	return &rateLimiter{interval: interval}
}

func (r *rateLimiter) wait(ctx context.Context) error {
	if r.interval <= 0 {
		return nil
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.last.IsZero() {
		r.last = time.Now()
		return nil
	}
	next := r.last.Add(r.interval)
	wait := time.Until(next)
	if wait <= 0 {
		r.last = time.Now()
		return nil
	}
	timer := time.NewTimer(wait)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		r.last = time.Now()
		return nil
	}
}

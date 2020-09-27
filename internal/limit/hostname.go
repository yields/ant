package limit

import (
	"context"
	"net/url"

	"golang.org/x/time/rate"
)

// Hostname implements a hostname limiter.
//
// When the configured hostname is seen the limiter
// will block until the URL is allowed to be fetched.
type Hostname struct {
	host  string
	limit *rate.Limiter
}

// ByHostname returns a new hostname limiter.
func ByHostname(host string, n int) *Hostname {
	return &Hostname{
		host:  host,
		limit: rate.NewLimiter(rate.Limit(n), n),
	}
}

// Limit implementation.
func (h *Hostname) Limit(ctx context.Context, u *url.URL) error {
	if u.Host == h.host {
		return h.limit.Wait(ctx)
	}
	return nil
}

package limit

import (
	"context"
	"net/url"

	"golang.org/x/time/rate"
)

// Limiter implements a global limiter.
//
// The limiter limits all URLs to `n` per second.
type Limiter struct {
	limit *rate.Limiter
}

// New returns a new limiter.
func New(n int) *Limiter {
	return &Limiter{
		limit: rate.NewLimiter(rate.Limit(n), n),
	}
}

// Limit implementation.
func (l *Limiter) Limit(ctx context.Context, _ *url.URL) error {
	return l.limit.Wait(ctx)
}

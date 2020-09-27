package limit

import (
	"context"
	"net/url"

	"github.com/tidwall/match"
	"golang.org/x/time/rate"
)

// Matcher implements a match limit.
//
// The limiter limits all URLs that match
// the provided pattern.
type Matcher struct {
	pattern string
	limit   *rate.Limiter
}

// ByMatch returns a new matcher limiter.
func ByMatch(pattern string, n int) *Matcher {
	return &Matcher{
		pattern: pattern,
		limit:   rate.NewLimiter(rate.Limit(n), n),
	}
}

// Limit implementation.
func (m *Matcher) Limit(ctx context.Context, u *url.URL) error {
	if match.Match(m.pattern, u.String()) {
		return m.limit.Wait(ctx)
	}
	return nil
}

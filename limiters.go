package ant

import (
	"context"
	"net/url"

	"github.com/tidwall/match"
	"golang.org/x/time/rate"
)

// Limiter controls how many requests can
// be made by the engine.
//
// A limiter receives a context and a URL and
// blocks until a request is allowed to happen
// or returns an error if the context is canceled.
type Limiter interface {
	// Limit blocks until a request is allowed to happen.
	//
	// The method receives a URL and must block until a request
	// to the URL is allowed to happen.
	//
	// If the given context is canceled, the method returns immediately
	// with the context's err.
	Limit(ctx context.Context, u *url.URL) error
}

// LimiterFunc implements a limiter.
type LimiterFunc func(context.Context, *url.URL) error

// Limit implementation.
func (f LimiterFunc) Limit(ctx context.Context, u *url.URL) error {
	return f(ctx, u)
}

// LimitHostname returns a hostname limiter.
//
// The limiter allows `n` requests for the hostname
// per second.
func LimitHostname(n int, name string) LimiterFunc {
	var limiter = rate.NewLimiter(rate.Limit(n), n)

	return func(ctx context.Context, u *url.URL) error {
		if u.Host == name {
			return limiter.Wait(ctx)
		}
		return nil
	}
}

// LimitMatch returns a match limiter.
//
// The limiter allows `n` requests for any URLs
// that match the pattern per second.
//
// The provided pattern is matched against a URL
// that does not contain the query string or the scheme.
func LimitMatch(n int, pattern string) LimiterFunc {
	var limiter = rate.NewLimiter(rate.Limit(n), n)

	return func(ctx context.Context, u *url.URL) error {
		var uri = u.Host + u.Path

		if match.Match(pattern, uri) {
			return limiter.Wait(ctx)
		}

		return nil
	}
}

// Limit returns a new limiter.
//
// The limiter allows `n` requests per second.
func Limit(n int) LimiterFunc {
	var limiter = rate.NewLimiter(rate.Limit(n), n)

	return func(ctx context.Context, _ *url.URL) error {
		return limiter.Wait(ctx)
	}
}

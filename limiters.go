package ant

import (
	"context"
	"fmt"
	"net/url"
	"regexp"

	"github.com/tidwall/match"
	"golang.org/x/time/rate"
)

// Limiter controls how many requests can be made by the engine.
//
// A limiter receives a context and a URL and
// blocks until a request is allowed to happen
// or returns an error if the context is canceled.
//
// A limiter must be safe to use from multiple goroutines.
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
	l := rate.NewLimiter(rate.Limit(n), n)
	return func(ctx context.Context, u *url.URL) error {
		if u.Host == name {
			return l.Wait(ctx)
		}
		return nil
	}
}

// LimitPattern returns a pattern limiter.
//
// The limiter allows `n` requests for any URLs
// that match the pattern per second.
//
// The provided pattern is matched against a URL
// that does not contain the query string or the scheme.
func LimitPattern(n int, pattern string) LimiterFunc {
	l := rate.NewLimiter(rate.Limit(n), n)
	return func(ctx context.Context, u *url.URL) error {
		if match.Match(u.Host+u.Path, pattern) {
			return l.Wait(ctx)
		}
		return nil
	}
}

// LimitRegexp returns a new regexp limiter.
//
// The limiter limits all URLs that match the regexp
// the URL does not contain the scheme and the query parameters.
func LimitRegexp(n int, expr string) LimiterFunc {
	l := rate.NewLimiter(rate.Limit(n), n)

	re, err := regexp.Compile(expr)
	if err != nil {
		panic(fmt.Sprintf("ant: regexp %q - %s", expr, err))
	}

	return func(ctx context.Context, u *url.URL) error {
		if re.MatchString(u.Host + u.Path) {
			return l.Wait(ctx)
		}
		return nil
	}
}

// Limit returns a new limiter.
//
// The limiter allows `n` requests per second.
func Limit(n int) LimiterFunc {
	l := rate.NewLimiter(rate.Limit(n), n)
	return func(ctx context.Context, _ *url.URL) error {
		return l.Wait(ctx)
	}
}

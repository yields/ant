package ant

import (
	"context"
	"net/url"

	"github.com/yields/ant/internal/limit"
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

// LimitHostname returns a hostname limiter.
//
// The limiter allows `n` requests for the hostname
// per second.
func LimitHostname(name string, n int) Limiter {
	return limit.ByHostname(name, n)
}

// LimitMatch returns a match limiter.
//
// The limiter allows `n` requests for any URLs
// that match the pattern per second.
func LimitMatch(pattern string, n int) Limiter {
	return limit.ByMatch(pattern, n)
}

// Limit returns a new limiter.
//
// The limiter allows `n` requests per second.
func Limit(n int) Limiter {
	return limit.New(n)
}

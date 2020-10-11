package antcache

import (
	"net/http"
	"time"
)

// Aggressive returns an aggressive cache with age.
//
// Unlike the default RFC7234 cache, this caching strategy
// will cache all GET/HEAD requests unless they specify
// a Range/Content-Range headers, the Authorization header
// and the "no-store" cache directive.
//
// This makes the aggressive cache arguably better for crawling
// since it can cache responses up to a specific age but still
// allows you to bypass the cache if you set the "no-cache" directive.
//
// The cacher will also ignore any "no-cache" and "no-store" or other
// directives from the response, since some websites never implement
// proper caching.
//
// When age <= 0, the default of 24 hours is used.
func Aggressive(age time.Duration) Option {
	return func(c *Cache) error {
		c.strategy = aggressive{age}
		return nil
	}
}

// Aggressive implements aggressive cache.
type aggressive struct {
	age time.Duration
}

// Cache implementation.
func (a aggressive) cache(req *http.Request) bool {
	return rfc7234{}.cache(req)
}

// Store implementation.
func (a aggressive) store(resp *http.Response) bool {
	var req = resp.Request

	// The request method is cacheable.
	switch req.Method {
	case "GET":
	case "HEAD":
	default:
		return false
	}

	// The response status code is cacheable.
	switch resp.StatusCode {
	case 200, 203, 204, 206:
	case 300, 301:
	case 404, 405, 410, 414:
	case 501:
	default:
		return false
	}

	// The response has a date header.
	_, ok := date(resp.Header)
	return ok
}

// Fresh implementation.
func (a aggressive) fresh(resp *http.Response) freshness {
	if date, ok := date(resp.Header); ok {
		if time.Since(date) < a.lifetime() {
			return fresh
		}
	}
	return transparent
}

// Lifetime returns the lifetime.
func (a aggressive) lifetime() time.Duration {
	if a.age > 0 {
		return a.age
	}
	return 24 * time.Hour
}

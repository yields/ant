package antcache

import (
	"net/http"
	"time"
)

// RFC7234 uses an RFC7234 caching implementation.
//
// Note that requests with content-range, range and authorization
// headers are never cached, some directives are also not
// implemented and are skipped ("immutable", "stale-if-error").
func RFC7234() Option {
	return func(c *Cache) error {
		c.strategy = rfc7234{}
		return nil
	}
}

// RFC7234 implements the standard cache strategy.
//
// https://tools.ietf.org/html/rfc7234
type rfc7234 struct{}

// Cache implementation.
//
// The method returns true if the request may use a cached
// response, or if it allows caching.
func (rfc7234) cache(req *http.Request) bool {
	return (req.Method == "GET" || req.Method == "HEAD") && !nostore(req.Header)
}

// Store implementation.
//
// https://tools.ietf.org/html/rfc7234#section-3
func (rfc7234) store(resp *http.Response) bool {
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

	// Parse request and response directives.
	var (
		reqd = directivesFrom(req.Header)
		resd = directivesFrom(resp.Header)
	)

	// the "no-store" cache directive (see Section 5.2) does not appear
	// in request or response header fields.
	if reqd.has("no-store") || resd.has("no-store") {
		return false
	}

	// ensure that the date header is set and
	// there's explicit expiry max-age/expires.
	if d, ok := date(resp.Header); ok {
		if maxage, ok := resd.duration("max-age"); ok {
			return maxage > 0
		}

		if exp, ok := expires(resp.Header); ok {
			return exp.Sub(d) > 0
		}
	}

	// The response has an explicit "lifetime" duration.
	return false
}

// Fresh implementation.
//
// https://tools.ietf.org/html/rfc7234#section-4
func (rfc7234) fresh(resp *http.Response) freshness {
	var req = resp.Request

	// selecting header fields nominated by the stored response (if any)
	// match those presented (see Section 4.1).
	if !matches(req, resp) {
		return transparent
	}

	// Parse request and response directives.
	var (
		reqd = directivesFrom(req.Header)
		resd = directivesFrom(resp.Header)
	)

	// the presented request does not contain the no-cache pragma
	// (Section 5.4), nor the no-cache cache directive (Section 5.2.1),
	// unless the stored response is successfully validated (Section 4.3).
	//
	// the stored response does not contain the no-cache cache directive
	// (Section 5.2.2.2), unless it is successfully validated (Section 4.3)
	if reqd.has("no-cache") || resd.has("no-cache") {
		return stale
	}

	// When only-if-cached is set, always return fresh.
	if reqd.has("only-if-cached") {
		return fresh
	}

	// the stored response is either fresh (see Section 4.2).
	if d, ok := date(resp.Header); ok {
		var age = time.Since(d)
		var lifetime time.Duration

		if maxage, ok := resd.duration("max-age"); ok {
			lifetime = maxage
		} else if exp, ok := expires(resp.Header); ok {
			lifetime = exp.Sub(d)
		}

		if maxage, ok := reqd.duration("max-age"); ok {
			lifetime = maxage
		}

		if minfresh, ok := reqd.duration("min-fresh"); ok {
			age += minfresh
		}

		if reqd.has("max-stale") {
			ms, ok := reqd.duration("max-stale")

			if !ok {
				return fresh
			}

			age -= ms
		}

		if lifetime > age {
			return fresh
		}
	}

	// validate (see Section 4.3).
	return stale
}

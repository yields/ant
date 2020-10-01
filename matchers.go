package ant

import (
	"net/url"
)

// Matcher represents a URL matcher.
type Matcher interface {
	// Match returns true if the URL matches.
	//
	// The method will be just before a URL is queued
	// if it returns false, the URL will not be queued.
	Match(url *url.URL) bool
}

// MatcherFunc implements a Matcher.
type MatcherFunc func(*url.URL) bool

// Match implementation.
func (mf MatcherFunc) Match(url *url.URL) bool {
	return mf(url)
}

// MatchHostname returns a new hostname matcher.
//
// The matcher returns true for all URLs that match the host.
func MatchHostname(host string) MatcherFunc {
	return func(url *url.URL) bool {
		return url.Host == host
	}
}

package ant

import (
	"fmt"
	"net/url"
	"regexp"

	"github.com/tidwall/match"
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

// MatchPattern returns a new pattern matcher.
//
// The matcher returns true for all URLs that match
// the pattern, the URL does not contain the scheme
// and the query parameters.
func MatchPattern(pattern string) MatcherFunc {
	return func(url *url.URL) bool {
		return match.Match(url.Host+url.Path, pattern)
	}
}

// MatchRegexp returns a new regexp matcher.
//
// The matcher returns true for all URLs that match
// the regexp, the URL does not contain the scheme
// and the query parameters.
func MatchRegexp(expr string) MatcherFunc {
	re, err := regexp.Compile(expr)
	if err != nil {
		panic(fmt.Sprintf("ant: regexp %q - %s", expr, err))
	}
	return func(url *url.URL) bool {
		fmt.Println("match", expr, url)
		return re.MatchString(url.Host + url.Path)
	}
}

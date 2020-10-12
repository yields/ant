package antcache

import (
	"net/http"
	"strings"
	"time"

	"github.com/spaolacci/murmur3"
)

// Merge merges b headers into a.
func merge(a http.Header, b http.Header) {
	for key := range b {
		switch key {
		case "Keep-Alive":
		case "Proxy-Authenticate":
		case "Proxy-Authorization":
		case "Te":
		case "Trailers":
		case "Transfer-Encoding":
		case "Upgrade":

		default:
			a[key] = b[key]
		}
	}
}

// Keyof returns a cache key of req.
func keyof(req *http.Request) uint64 {
	return murmur3.Sum64([]byte(
		req.Method + ":" + req.URL.String(),
	))
}

// Matches ensures that the given request and response match.
//
// https://tools.ietf.org/html/rfc7234#section-4.1
func matches(req *http.Request, resp *http.Response) bool {
	var vary = req.Header.Get("Vary")

	for _, h := range split(vary, ",") {
		if key := http.CanonicalHeaderKey(h); key != "" {
			if req.Header.Get(key) != resp.Header.Get(key) {
				return false
			}
		}
	}

	return true
}

// Nostore returns true if no-store is set.
func nostore(h http.Header) bool {
	var c = h.Get("Cache-Control")

	for _, v := range split(c, ",") {
		if v == "no-store" {
			return true
		}
	}

	return false
}

// Expires returns the expires timestamp.
//
// When expires does not exist or is zero, ok is false.
func expires(h http.Header) (expires time.Time, ok bool) {
	if v := h.Get("Expires"); v != "" {
		t, err := time.Parse(time.RFC1123, v)
		expires, ok = t, (err == nil && !t.IsZero())
	}
	return
}

// Date returns the date timestamp.
//
// When date does not exist or is zero, ok is false.
func date(h http.Header) (date time.Time, ok bool) {
	if v := h.Get("Date"); v != "" {
		t, err := time.Parse(time.RFC1123, v)
		date, ok = t, (err == nil && !t.IsZero())
	}
	return
}

// Split splits the given str by sep.
//
// The method omits any empty values and normalizes
// the values by lowercasing them.
func split(str, sep string) (ret []string) {
	for _, v := range strings.Split(str, sep) {
		if v := strings.TrimSpace(v); v != "" {
			ret = append(ret, strings.ToLower(v))
		}
	}
	return
}

// Package ant implements a web crawler.
package ant

import (
	"context"
	"net"
	"net/http"
	"net/url"
	"time"
)

// URL represents a parsed URL.
type URL = url.URL

// URLs represents a slice of parsed URLs.
type URLs = []*URL

// Scraper represents a scraper.
//
// A scraper must be safe to use from multiple goroutines.
type Scraper interface {
	// Scrape scrapes the given page.
	//
	// The method can return a set of URLs that should
	// be queued and scraped next.
	//
	// If the scraper returns an error and it implements
	// a `Temporary() bool` method that returns true it will
	// be retried.
	Scrape(ctx context.Context, p *Page) (URLs, error)
}

// Client represents an HTTP client.
//
// A client is used by the fetcher to turn URLs into pages, it is up to
// the client to decide how it manages the underlying connections, redirects
// or cookies.
//
// A client must be safe to use from multiple goroutines.
type Client interface {
	// Do sends an HTTP request and returns an HTTP response.
	//
	// The method does not rely on the HTTP response code to return an error
	// also a non-nil error does not guarantee that the response is nil, its
	// body must be closed and read until EOF so that the underlying resources
	// may be reused.
	Do(req *http.Request) (*http.Response, error)
}

// DefaultClient is the default client to use.
//
// It is configured the same way as the `http.DefualtClient`
// except for 3 changes:
//
//  - Timeout                       => 10s
//  - Transport.MaxIdleConns        => infinity
//  - Transport.MaxIdleConnsPerHost => 1,000
//
// Note that this default client is used for all robots.txt
// requests when they're enabled.
var DefaultClient = &http.Client{
	Transport: &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
			DualStack: true,
		}).DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          0,    // was 100.
		MaxIdleConnsPerHost:   1000, // was 2.
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	},
	Timeout: 10 * time.Second,
}

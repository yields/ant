// Package ant implements a web crawler.
package ant

import (
	"context"
	"net/url"
)

// URL represents a parsed URL.
type URL = url.URL

// URLs represents a slice of parsed URLs.
type URLs = []*URL

// Scraper represents a scraper.
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

// Fetcher represents a page fetcher.
type Fetcher interface {
	// Fetch fetches a page using a url.
	//
	// If the fetcher returns an error that implements
	// `Temporary() bool` that returns true the engine
	// will retry fetching the URL.
	Fetch(ctx context.Context, url *URL) (*Page, error)
}

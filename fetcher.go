package ant

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
)

// StaticAgent is a static user agent string.
type StaticAgent string

// String implementation.
func (sa StaticAgent) String() string {
	return string(sa)
}

var (
	// UserAgent is the default user agent to use.
	//
	// The user agent is used by default when fetching
	// pages and robots.txt.
	UserAgent = StaticAgent("antbot")

	// DefaultFetcher is the default fetcher to use.
	//
	// It uses the default client and default user agent.
	DefaultFetcher = &Fetcher{
		Client:    DefaultClient,
		UserAgent: UserAgent,
	}
)

// Fetch fetches a page from URL.
func Fetch(ctx context.Context, rawurl string) (*Page, error) {
	u, err := url.Parse(rawurl)
	if err != nil {
		return nil, err
	}
	return DefaultFetcher.fetch(ctx, u)
}

// Fetcher implements a page fetcher.
type Fetcher struct {
	// Client is the client to use.
	//
	// If nil, ant.DefaultClient is used.
	Client Client

	// UserAgent is the user agent to use.
	//
	// It implements the fmt.Stringer interface
	// to allow user agent spoofing when needed.
	//
	// If nil, the client decides the user agent.
	UserAgent fmt.Stringer
}

// Fetch fetches a new page by URL.
func (f *Fetcher) fetch(ctx context.Context, url *URL) (*Page, error) {
	var client = f.client()

	req, err := http.NewRequestWithContext(ctx, "GET", url.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("ant: new request - %w", err)
	}

	for k, v := range f.headers() {
		req.Header[k] = v
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ant: %s %q - %w", req.Method, req.URL, err)
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("ant: %s %q - %s",
			resp.Request.Method,
			resp.Request.URL,
			resp.Status,
		)
	}

	return &Page{
		URL:  resp.Request.URL,
		body: resp.Body,
	}, nil
}

// Headers returns all headers.
func (f *Fetcher) headers() http.Header {
	var hdr = make(http.Header)

	hdr.Set("Accept", "text/html; charset=UTF-8")

	if ua := f.UserAgent; ua != nil {
		hdr.Set("User-Agent", ua.String())
	}

	return hdr
}

// Client returns the client to use.
func (f *Fetcher) client() Client {
	if f.Client != nil {
		return f.Client
	}
	return DefaultClient
}

package ant

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"time"
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

	// MinBackoff to use when the fetcher retries.
	//
	// Must be less than MaxBackoff, otherwise
	// the fetcher returns an error.
	minBackoff = 50 * time.Millisecond

	// MaxBackoff to use when the fetcher retries.
	//
	// Must be greater than MinBackoff, otherwise the
	// fetcher returns an error.
	maxBackoff = 1 * time.Second
)

// FetchError represents a fetch error.
type FetchError struct {
	URL    *url.URL
	Status int
}

// Error implementation.
func (err FetchError) Error() string {
	return fmt.Sprintf("ant: fetch %q - %d %s",
		err.URL,
		err.Status,
		http.StatusText(err.Status),
	)
}

// Temporary returns true if the HTTP status code
// generally means the error is temporary.
func (err FetchError) Temporary() bool {
	return err.Status == 503 || // Service Unavailable.
		err.Status == 504 || // Gateway Timeout.
		err.Status == 429 // Too many requests.
}

// Fetch fetches a page from URL.
func Fetch(ctx context.Context, rawurl string) (*Page, error) {
	u, err := url.Parse(rawurl)
	if err != nil {
		return nil, err
	}
	return DefaultFetcher.Fetch(ctx, u)
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

	// MaxAttempts is the maximum request attempts to make.
	//
	// When <= 0, it defaults to 5.
	MaxAttempts int

	// MinBackoff to use when the fetcher retries.
	//
	// Must be less than MaxBackoff, otherwise
	// the fetcher returns an error.
	//
	// Defaults to `50ms`.
	MinBackoff time.Duration

	// MaxBackoff to use when the fetcher retries.
	//
	// Must be greater than MinBackoff, otherwise the
	// fetcher returns an error.
	//
	// Defaults to `1s`.
	MaxBackoff time.Duration
}

// Fetch fetches a page by URL.
//
// The method uses the configured client to make a new request
// parse the response and return a page.
//
// The method returns a nil page and nil error when the status
// code is 404.
//
// The will retry the request when the status code is temporary
// or when a temporary network error occures.
//
// The returned page contains the response's body, the body must
// be read until EOF and closed so that the client can re-use the
// underlying TCP connection.
func (f *Fetcher) Fetch(ctx context.Context, url *URL) (*Page, error) {
	var maxAttempts = f.maxAttempts()
	var attempt int
	var resp *http.Response
	var err error

	for {
		if attempt++; attempt > maxAttempts {
			return nil, fmt.Errorf(
				"ant: max attempts of %d reached - %w",
				maxAttempts,
				err,
			)
		}

		if resp, err = f.fetch(ctx, url); err == nil {
			break
		}

		f.discard(resp)
		if isTemporary(err) {
			if err := f.backoff(ctx, attempt); err != nil {
				return nil, err
			}
			continue
		}

		if err, ok := err.(*FetchError); ok {
			if err.Status == 404 {
				return nil, nil
			}
		}

		return nil, err
	}

	return &Page{
		URL:    resp.Request.URL,
		Header: resp.Header,
		body:   resp.Body,
	}, nil
}

// Fetch fetches a new page by URL.
func (f *Fetcher) fetch(ctx context.Context, url *URL) (*http.Response, error) {
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
		return resp, fmt.Errorf("ant: %s %q - %w", req.Method, req.URL, err)
	}

	if resp.StatusCode >= 400 {
		return resp, &FetchError{
			URL:    resp.Request.URL,
			Status: resp.StatusCode,
		}
	}

	return resp, nil
}

// Discard discards the given response.
func (f *Fetcher) discard(r *http.Response) {
	if r != nil {
		io.Copy(ioutil.Discard, r.Body)
		r.Body.Close()
	}
}

// MaxAttempts returns the max attempts.
func (f *Fetcher) maxAttempts() int {
	if f.MaxAttempts > 0 {
		return f.MaxAttempts
	}
	return 5
}

// Headers returns all headers.
func (f *Fetcher) headers() http.Header {
	var hdr = make(http.Header)

	hdr.Set("Accept", "text/html; charset=UTF-8")
	hdr.Set("User-Agent", f.userAgent())

	return hdr
}

// UserAgent returns the user agent to use.
func (f *Fetcher) userAgent() string {
	if ua := f.UserAgent; ua != nil {
		return ua.String()
	}
	return UserAgent.String()
}

// Client returns the client to use.
func (f *Fetcher) client() Client {
	if f.Client != nil {
		return f.Client
	}
	return DefaultClient
}

// Backoff performs the backoff.
//
// TODO: configurable backoff duration, jitter...?
func (f *Fetcher) backoff(ctx context.Context, attempt int) error {
	var min = f.minBackoff()
	var max = f.maxBackoff()
	var dur = time.Duration(attempt*attempt) * min

	if min >= max {
		return fmt.Errorf("ant: min backoff must be greater than max backoff")
	}

	if dur > max {
		dur = max
	}

	var timer = time.NewTimer(dur)
	defer timer.Stop()

	select {
	case <-timer.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// MinBackoff returns the min backoff.
func (f *Fetcher) minBackoff() time.Duration {
	if f.MinBackoff > 0 {
		return f.MinBackoff
	}
	return minBackoff
}

// MaxBackoff returns the min backoff.
func (f *Fetcher) maxBackoff() time.Duration {
	if f.MaxBackoff > 0 {
		return f.MaxBackoff
	}
	return maxBackoff
}

// IsTemporary returns true if the error is temporary.
func isTemporary(err error) bool {
	t, ok := err.(interface{ Temporary() bool })
	return ok && t.Temporary()
}

package ant

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"time"
)

var (
	// UserAgent is the user agent that ant sends  by default.
	UserAgent = "ant/1"

	// Client is the default http client to use.
	//
	// It is configured the same way as the `http.DefaultClient`
	// except for 3 changes:
	//
	//  - MaxIdleConns => 0
	//  - MaxIdleConnsPerHost => 1000
	//  - Timeout => 10
	//
	Client = &http.Client{
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
)

// HTTP implements an HTTP page fetcher.
type HTTP struct {
	// UserAgent is a user agent func.
	//
	// If empty, `ant.UserAgent` is used instead.
	UserAgent string

	// Prepare is a function to prepare a request.
	//
	// The function is called before every request.
	Prepare func(*http.Request)

	// Client is the HTTP client to use.
	//
	// If empty, `http.DefaultClient` is used instead.
	Client *http.Client
}

// HttpError represents an http error.
type httpError struct {
	status int
	url    string
}

// Error implementation.
func (err httpError) Error() string {
	return fmt.Sprintf("ant: GET %q - %d %s",
		err.url,
		err.status,
		http.StatusText(err.status),
	)
}

// Skip implementation.
func (err httpError) Skip() bool {
	return err.status == 404 ||
		err.status == 406 ||
		err.status == 405
}

// Temporary implementation.
func (err httpError) Temporary() bool {
	return err.status == 500 ||
		err.status == 503
}

// Fetch fetches a page at URL.
func Fetch(ctx context.Context, url string) (*Page, error) {
	return (HTTP{}).Fetch(ctx, url)
}

// Fetch implementation.
func (h HTTP) Fetch(ctx context.Context, url string) (*Page, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("ant: new request - %w", err)
	}

	req.Header.Set("User-Agent", h.userAgent())
	req.Header.Set("Accept-Language", "en-US")
	req.Header.Set("Accept", "text/html")

	if h.Prepare != nil {
		h.Prepare(req)
	}

	resp, err := h.client().Do(req)

	if err != nil {
		return nil, fmt.Errorf("ant: request %q - %w", url, err)
	}

	if resp.StatusCode >= 400 {
		resp.Body.Close()
		return nil, httpError{
			status: resp.StatusCode,
			url:    resp.Request.URL.String(),
		}
	}

	return &Page{
		URL:  resp.Request.URL,
		body: resp.Body,
	}, nil
}

// UserAgent returns the user agent.
func (h HTTP) userAgent() string {
	if h.UserAgent != "" {
		return h.UserAgent
	}
	return UserAgent
}

// Client returns the client to use.
func (h HTTP) client() *http.Client {
	if h.Client != nil {
		return h.Client
	}
	return Client
}

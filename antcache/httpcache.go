// Package antcache implements an HTTP client that caches responses.
package antcache

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
)

// Freshness enumerates freshness.
type freshness int

// String implementation.
func (f freshness) String() string {
	switch f {
	case fresh:
		return "fresh"
	case stale:
		return "stale"
	case transparent:
		return "transprent"
	default:
		return fmt.Sprintf("antcache.freshness(%d)", f)
	}
}

// All freshness types.
const (
	fresh freshness = iota
	stale
	transparent
)

// Strategy represents a cache strategy.
type strategy interface {
	// Cache returns true if the request is cacheable.
	//
	// The method is called just before a the storage lookup
	// is made.
	cache(req *http.Request) bool

	// Store returns true if the response can be stored.
	//
	// The method is called just before a response is stored.
	store(resp *http.Response) bool

	// Fresh returns true if the response is fresh.
	//
	// The method is called just before a cached response
	// is returned from the cache.
	fresh(resp *http.Response) freshness
}

// Storage represents the cache storage.
//
// A storage must be safe to use from multiple goroutines.
type Storage interface {
	// Store stores the given response.
	//
	// The method is called just after the response's body is
	// closed, the value contains the full response including headers.
	Store(ctx context.Context, key uint64, value []byte) error

	// Load loads a response by its key.
	//
	// When the response is not found, the method returns a nil
	// byteslice and a nil error.
	//
	// The method returns the full response, as stored by `Store()`.
	Load(ctx context.Context, key uint64) ([]byte, error)
}

// Client represents an HTTP client.
type Client interface {
	// Do performs the given request.
	Do(req *http.Request) (*http.Response, error)
}

// Option represents a cache option.
type Option func(*Cache) error

// WithStorage sets the storage to s.
func WithStorage(s Storage) Option {
	return func(c *Cache) error {
		if s == nil {
			return errors.New("antcache: storage must be non-nil")
		}
		c.storage = s
		return nil
	}
}

// Cache implements an HTTP cache.
type Cache struct {
	storage  Storage
	strategy strategy
	client   Client
}

// New returns a new cache with the given options.
func New(c Client, opts ...Option) (*Cache, error) {
	var cache = &Cache{
		strategy: rfc7234{},
		storage:  &memstore{},
		client:   c,
	}

	if c == nil {
		return nil, errors.New("antcache: client must be non-nil")
	}

	for _, opt := range opts {
		if err := opt(cache); err != nil {
			return nil, err
		}
	}

	return cache, nil
}

// Do performs the given request.
//
// The method initially checks if the request can be cached
// if so, it will lookup a response that matches the request
// and return it if it's fresh, or was validated.
//
// When the request is not cacheable, the method simply calls
// the underlying client with the request and never caches the response.
//
// When a response is not found, the method calls the underlying client
// and if the response can be stored, it will store it when its body
// is closed, if the body is not closed, the response is never stored.
//
// If there was an error loading a cached response the method returns
// the error and discards the response's body, if an error occurs
// when storing the response body, the response's Close() method
// will return the error.
func (c *Cache) Do(req *http.Request) (*http.Response, error) {
	if !c.strategy.cache(req) {
		return c.client.Do(req)
	}

	var key = keyof(req)

	resp, err := c.load(key, req)
	if err != nil {
		return nil, err
	}
	if resp != nil {
		resp.Header.Set("X-From-Cache", "1")
		return resp, nil
	}

	resp, err = c.client.Do(req)
	if err != nil {
		return nil, err
	}

	if c.strategy.store(resp) {
		c.store(key, resp)
	}

	return resp, nil
}

// Load loads a response from the cache storage for req.
//
// The method attempts to load a cached response for req, if a response
// is found the response is checked to ensure it is not stale, if it's fresh
// it is immediately returned, the method will verify stale requests as needed.
//
// When a response is refreshed from the origin server it will be overwritten
// in the storage once the response's body is closed.
//
// The method returns nil response and nil error when the response does not
// exist in the cache or when it must be refreshed.
func (c *Cache) load(key uint64, req *http.Request) (*http.Response, error) {
	var ctx = req.Context()

	buf, err := c.storage.Load(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("antcache: load %d - %w", key, err)
	}
	if buf == nil {
		return nil, nil
	}

	b := bytes.NewBuffer(buf)
	r := bufio.NewReader(b)

	resp, err := http.ReadResponse(r, req)
	if err != nil {
		return nil, fmt.Errorf("antcache: read response %d - %w", key, err)
	}

	switch c.strategy.fresh(resp) {
	case fresh:
		return resp, nil

	case stale:
		return c.verify(ctx, key, resp)
	}

	return nil, nil
}

// Verify verifies that the given response is still valid.
//
// https://tools.ietf.org/html/rfc7234#section-4.3.
func (c *Cache) verify(ctx context.Context, key uint64, resp *http.Response) (*http.Response, error) {
	var req = resp.Request.Clone(ctx)
	var hdr = resp.Header

	if etag := hdr.Get("ETag"); etag != "" {
		if req.Header.Get("If-None-Match") == "" {
			req.Header.Set("If-None-Match", etag)
		}
	}

	if t := hdr.Get("Last-Modified"); t != "" {
		if req.Header.Get("If-Modified-Since") == "" {
			req.Header.Set("If-Modified-Since", t)
		}
	}

	newresp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("antcache: validate %d - %w", key, err)
	}

	// If a cache receives a 5xx (Server Error) response while
	// attempting to validate a response, it can either forward this
	// response to the requesting client, or act as if the server failed
	// to respond.  In the latter case, the cache MAY send a previously
	// stored response (see Section 4.2.4).
	if newresp.StatusCode >= 500 && newresp.StatusCode < 600 {
		reqd := directivesFrom(req.Header)
		if reqd.has("stale-if-error") {
			return resp, nil
		}
		return newresp, nil
	}

	// A 304 (Not Modified) response status code indicates that the
	// stored response can be updated and reused; see Section 4.3.4.
	if newresp.StatusCode == 304 {
		c.discard(resp)
		merge(resp.Header, newresp.Header)
		c.store(key, resp)
		return resp, nil
	}

	// A full response (i.e., one with a payload body) indicates that
	// none of the stored responses nominated in the conditional request
	// is suitable.  Instead, the cache MUST use the full response to
	// satisfy the request and MAY replace the stored response(s).
	if c.strategy.store(newresp) {
		c.discard(resp)
		c.store(key, newresp)
		return newresp, nil
	}

	// Verification failed, cleanup and close the response readers.
	c.discard(newresp)
	c.discard(resp)
	return nil, nil
}

// Store stores the given response.
//
// The method overwrites the response's body with a readcloser
// that will write the response to the storage when it is closed.
//
// If the response body is not closed, the response is never stored in the cache.
func (c *Cache) store(key uint64, resp *http.Response) {
	rc := resp.Body

	resp.Body = &cachereader{
		resp:  resp,
		key:   key,
		rc:    rc,
		ctx:   resp.Request.Context(),
		store: c.storage.Store,
	}

	return
}

// Discard discasrds the given reader.
func (c *Cache) discard(resp *http.Response) {
	if resp != nil {
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	}
}

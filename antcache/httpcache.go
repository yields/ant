// Package antcache implements an HTTP client that caches responses.
package antcache

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"sync"
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

// WithLogger sets the logger to log.
func WithLogger(log *log.Logger) Option {
	return func(c *Cache) error {
		if log == nil {
			return errors.New("antcache: log must be non-nil")
		}
		c.log = log
		return nil
	}
}

// Cache implements an HTTP cache.
type Cache struct {
	storage  Storage
	strategy strategy
	client   Client
	log      *log.Logger
}

// New returns a new cache with the given options.
func New(c Client, opts ...Option) (*Cache, error) {
	var cache = &Cache{
		strategy: rfc7234{},
		storage:  &memstore{},
		client:   c,
		log:      nil,
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
// is closed, if the body is not closed, the response is never cached.
//
// When storage errors occure while loading/storing a response the method
// logs the errors if a logger is set and no errors are returned, the method
// will simply fallback to the underlying client on storage errors.
func (c *Cache) Do(req *http.Request) (*http.Response, error) {
	if !c.strategy.cache(req) {
		return c.client.Do(req)
	}

	var key = keyof(req)

	if resp, ok := c.load(key, req); ok {
		return resp, nil
	}

	resp, err := c.client.Do(req)
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
// is found in the storage and is fresh the method returns the response
// along with `ok=true`.
//
// If any storage related errors occur, the method simply logs the errors
// and returns a nil response with ok=false, if an error logger is not
// defined on the cache, no logs are produced.
//
// If the response is stale, the method will send a validation request
// and if the response is still fresh, the method returns it and updates
// the cached response.
func (c *Cache) load(key uint64, req *http.Request) (*http.Response, bool) {
	var ctx = req.Context()

	buf, err := c.storage.Load(ctx, key)
	if err != nil {
		c.error("antcache: storage load %d - %s", key, err)
		return nil, false
	}

	b := bytes.NewBuffer(buf)
	r := bufio.NewReader(b)

	resp, err := http.ReadResponse(r, req)
	if err != nil {
		c.error("antcache: read cached response - %s", err)
		return nil, false
	}

	switch c.strategy.fresh(resp) {
	case fresh:
		return resp, true

	case stale:
		return c.verify(ctx, key, resp)
	}

	return nil, false
}

// Verify verifies that the given response is still valid.
func (c *Cache) verify(ctx context.Context, key uint64, resp *http.Response) (*http.Response, bool) {
	var req = resp.Request.Clone(ctx)
	var hdr = resp.Header

	if etag := hdr.Get("ETag"); etag != "" {
		req.Header.Set("If-None-Match", etag)
	}

	if t := hdr.Get("Last-Modified"); t != "" {
		req.Header.Set("If-Modified-Since", t)
	}

	newresp, err := c.client.Do(req)
	if err != nil {
		c.error("antcache: validate %s - %s", req.URL, err)
		return nil, false
	}

	if newresp.StatusCode == 304 {
		return resp, true
	}

	if c.strategy.store(resp) {
		c.store(key, newresp)
		return resp, true
	}

	return nil, false
}

// Store stores the given response.
//
// The method overwrites the response's body with a readcloser
// that will write the response to the storage when it is closed.
//
// If the response body is not closed, the response is never
// stored in the cache.
func (c *Cache) store(key uint64, resp *http.Response) {
	rc := resp.Body

	resp.Body = &cachereader{
		resp:  resp,
		key:   key,
		rc:    rc,
		buf:   &bytes.Buffer{},
		once:  &sync.Once{},
		ctx:   resp.Request.Context(),
		store: c.storage.Store,
		log:   c.error,
	}

	return
}

// Error logs an error with msg and args.
func (c *Cache) error(msg string, args ...interface{}) {
	if c.log != nil {
		c.log.Printf(msg, args...)
	}
}

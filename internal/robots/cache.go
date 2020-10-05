// Package robots implements a higher-level robots.txt interface.
//
// The package implements a cache that caches robots.txt structures
// per hostname.
package robots

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/segmentio/agecache"
	"github.com/temoto/robotstxt"
)

// Request represents a request.
//
// If the UserAgent is empty it will
// default to `*`.
type Request struct {
	UserAgent string
	URL       *url.URL
}

// UserAgent returns the useragent or *.
func (r Request) userAgent() string {
	if r.UserAgent != "" {
		return r.UserAgent
	}
	return "*"
}

// Host represents a host.
//
// The host contains the host's robots.txt structures.
type Host struct {
	data *robotstxt.RobotsData
}

// Find returns a group by useragent.
func (h *Host) find(ua string) (*robotstxt.Group, bool) {
	if h.data != nil {
		g := h.data.FindGroup(ua)
		return g, g != nil
	}
	return nil, false
}

// Test tests the useragent.
func (h *Host) test(path, ua string) bool {
	if h.data != nil {
		return h.data.TestAgent(path, ua)
	}
	return true
}

// Cache implements an LRU robots cache.
//
// The cache maintains an LRU of domain names
// into their robots.txt structures, when a new
// domain is seen the cache will fetch the robots.txt
// parse it, and add it to the cache.
type Cache struct {
	lru    *agecache.Cache
	client *http.Client
}

// NewCache returns a new cache with the client and cache capacity.
func NewCache(c *http.Client, capacity int) *Cache {
	lru := agecache.New(agecache.Config{
		Capacity:           capacity,
		MaxAge:             1 * time.Hour,
		ExpirationType:     agecache.PassiveExpration,
		ExpirationInterval: 1 * time.Minute,
	})
	return &Cache{lru: lru, client: c}
}

// Allowed returns true if the request is allowed.
//
// The method will lookup the robots.txt structure for
// the domain name and check if the request user agent
// is allowed to fetch the URL. Subsequent calls may use
// the cached robots.txt structures.
//
// The method returns an error if the context is canceled
// or if a parsing error occurs.
func (c *Cache) Allowed(ctx context.Context, req Request) (bool, error) {
	var path = req.URL.Path
	var ua = req.userAgent()

	if len(path) > 0 && path[0] != '/' {
		path = "/" + path
	}

	host, err := c.lookup(ctx, req.URL)
	if err != nil {
		return false, err
	}

	return host.test(path, ua), nil
}

// Wait blocks until the given request can be sent.
//
// Some robots.txt define a crawl delay for all or some of the useragents.
// The method will block until the request can go through.
func (c *Cache) Wait(ctx context.Context, req Request) error {
	var ua = req.userAgent()

	host, err := c.lookup(ctx, req.URL)
	if err != nil {
		return err
	}

	if g, ok := host.find(ua); ok {
		if d := g.CrawlDelay; d > 0 {
			t := time.NewTimer(d)
			defer t.Stop()
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-t.C:
				return nil
			}
		}
	}

	return nil
}

// Lookup returns a host from url.
//
// Note that there's a logical race, the method may send multiple requests
// for the same robots.txt URL, this is intentional to speed up lookups.
func (c *Cache) lookup(ctx context.Context, url *url.URL) (*Host, error) {
	if v, ok := c.lru.Get(url.Host); ok {
		return v.(*Host), nil
	}

	rawurl := url.Scheme + "://" + url.Host + "/robots.txt"
	req, err := http.NewRequestWithContext(ctx, "GET", rawurl, nil)
	if err != nil {
		return nil, fmt.Errorf("robots: new request - %w", err)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("robots: GET %q - %w", rawurl, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		s := &Host{}
		c.lru.Set(url.Host, s)
		return s, nil
	}

	data, err := robotstxt.FromResponse(resp)
	if err != nil {
		return nil, fmt.Errorf("robots: parse robots.txt - %w", err)
	}

	s := &Host{data: data}
	c.lru.Set(url.Host, s)
	return s, nil
}

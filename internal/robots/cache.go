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

// Site represents a site.
type Site struct {
	data *robotstxt.RobotsData
}

// Find returns a group by useragent.
func (s *Site) find(ua string) (*robotstxt.Group, bool) {
	if s.data != nil {
		g := s.data.FindGroup(ua)
		return g, g != nil
	}
	return nil, false
}

// Test tests the useragent.
func (s *Site) test(path, ua string) bool {
	if s.data != nil {
		return s.data.TestAgent(path, ua)
	}
	return true
}

// Cache implements an LRU robots cache.
//
// The cache maintains an LRU of domain names
// into their robots.txt structures, when a new
// domain is seen the cache will fetch the robots.txt
// parse it and add it to the cache.
type Cache struct {
	lru *agecache.Cache
}

// NewCache returns a new cache.
func NewCache(capacity int) *Cache {
	lru := agecache.New(agecache.Config{
		Capacity:           capacity,
		MaxAge:             1 * time.Hour,
		ExpirationType:     agecache.PassiveExpration,
		ExpirationInterval: 1 * time.Minute,
	})
	return &Cache{lru: lru}
}

// Allowed returns true if the request is allowed.
//
// The method will lookup the robots.txt structure for
// the domain name and check if the request user agent
// is allowed to fetch the URL.
//
// If the domain has not been seen before the method will
// automatically attempt to fetch the robots.txt and parse it.
//
// The method returns an error if the context is canceled
// or if a parsing error occurs.
func (c *Cache) Allowed(ctx context.Context, req Request) (bool, error) {
	var path = req.URL.Path
	var ua = req.userAgent()

	site, err := c.lookup(ctx, req.URL)
	if err != nil {
		return false, err
	}

	return site.test(path, ua), nil
}

// Wait blocks until the given request can be sent.
//
// Some robots.txt define a delay for all or some of the useragents
// the method will block until the request can go through.
func (c *Cache) Wait(ctx context.Context, req Request) error {
	var ua = req.userAgent()

	site, err := c.lookup(ctx, req.URL)
	if err != nil {
		return err
	}

	if g, ok := site.find(ua); ok {
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

// Lookup returns a site from url.
func (c *Cache) lookup(ctx context.Context, url *url.URL) (*Site, error) {
	if v, ok := c.lru.Get(url.Host); ok {
		return v.(*Site), nil
	}

	rawurl := url.Scheme + "://" + url.Host + "/robots.txt"
	req, err := http.NewRequestWithContext(ctx, "GET", rawurl, nil)
	if err != nil {
		return nil, fmt.Errorf("robots: new request - %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("robots: GET %q - %w", rawurl, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		s := &Site{}
		c.lru.Set(url.Host, s)
		return s, nil
	}

	data, err := robotstxt.FromResponse(resp)
	if err != nil {
		return nil, fmt.Errorf("robots: parse robots.txt - %w", err)
	}

	s := &Site{data: data}
	c.lru.Set(url.Host, s)
	return s, nil
}

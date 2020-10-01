package ant

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/url"
	"runtime"

	"github.com/yields/ant/internal/normalize"
	"github.com/yields/ant/internal/robots"
	"golang.org/x/sync/errgroup"
)

// EngineConfig configures the engine.
type EngineConfig struct {
	// Scraper is the scraper to use.
	//
	// If nil, NewEngine returns an error.
	Scraper Scraper

	// Deduper is the URL de-duplicator to use.
	//
	// If nil, DedupeMap is used.
	Deduper Deduper

	// Fetcher is the page fetcher to use.
	//
	// If nil, the default HTTP fetcher is used.
	Fetcher *Fetcher

	// Queue is the URL queue to use.
	//
	// If nil, the default in-memory queue is used.
	Queue Queue

	// Limiter is the rate limiter to use.
	//
	// The limiter is called with each URL before
	// it is fetched.
	//
	// If nil, no limits are used.
	Limiter Limiter

	// Matcher is the URL matcher to use.
	//
	// The matcher is called with a URL before it is queued
	// if it returns false the URL is discarded.
	//
	// If nil, all URLs are queued.
	Matcher Matcher

	// Concurrency controls the amount of goroutines
	// the engine starts.
	//
	// Every goroutine is in charge of fetching a page
	// calling the scraper and enqueueing the urls
	// the scraper has returned.
	//
	// If <= 0, it defaults to runtime.GOMAXPROCS.
	Concurrency int
}

// Engine implements web crawler engine.
type Engine struct {
	deduper     Deduper
	scraper     Scraper
	fetcher     *Fetcher
	queue       Queue
	matcher     Matcher
	limiter     Limiter
	robots      *robots.Cache
	concurrency int
}

// NewEngine returns a new engine.
func NewEngine(c EngineConfig) (*Engine, error) {
	if c.Scraper == nil {
		return nil, errors.New("ant: scraper is required")
	}

	if c.Deduper == nil {
		c.Deduper = DedupeMap()
	}

	if c.Fetcher == nil {
		c.Fetcher = &Fetcher{}
	}

	if c.Concurrency <= 0 {
		c.Concurrency = runtime.GOMAXPROCS(-1)
	}

	if c.Queue == nil {
		c.Queue = MemoryQueue(c.Concurrency)
	}

	return &Engine{
		scraper:     c.Scraper,
		deduper:     c.Deduper,
		fetcher:     c.Fetcher,
		queue:       c.Queue,
		matcher:     c.Matcher,
		limiter:     c.Limiter,
		robots:      robots.NewCache(1000),
		concurrency: c.Concurrency,
	}, nil
}

// Run runs the engine with the given start urls.
func (eng *Engine) Run(ctx context.Context, urls ...string) error {
	var eg, subctx = errgroup.WithContext(ctx)

	// Spawn workers.
	//
	// TODO: probably need to spawn as needed instead of pooling.
	for i := 0; i < eng.concurrency; i++ {
		eg.Go(func() error {
			defer eng.queue.Close()
			return eng.run(subctx)
		})
	}

	// Enqueue initial URLs.
	if err := eng.Enqueue(ctx, urls...); err != nil {
		return fmt.Errorf("ant: enqueue - %w", err)
	}

	// Wait until all URLs are handled.
	eng.queue.Wait()
	if err := eng.queue.Close(); err != nil {
		return err
	}

	// Wait until all workers shutdown.
	if err := eg.Wait(); err != nil {
		return fmt.Errorf("ant: run - %w", err)
	}

	return nil
}

// Enqueue enqueues the given set of URLs.
//
// The method blocks until all URLs are queued
// or the given context is canceled.
//
// The method will also de-duplicate the URLs, ensuring
// that URLs will not be visited more than once.
func (eng *Engine) Enqueue(ctx context.Context, rawurls ...string) error {
	var batch = make(URLs, 0, len(rawurls))

	for _, rawurl := range rawurls {
		u, err := url.Parse(rawurl)
		if err != nil {
			return fmt.Errorf("ant: parse url %q - %w", rawurl, err)
		}
		batch = append(batch, u)
	}

	return eng.enqueue(ctx, batch)
}

// Enqueue enqueues the given parsed urls.
func (eng *Engine) enqueue(ctx context.Context, batch URLs) error {
	for j := range batch {
		batch[j] = normalize.URL(batch[j])
	}

	next, err := eng.dedupe(ctx, eng.matches(batch))
	if err != nil {
		return err
	}

	if err := eng.queue.Enqueue(ctx, next); err != nil {
		return err
	}

	return nil
}

// Run runs a single crawl worker.
//
// The worker is in charge of fetching a url from
// the queue, creating a page and then calling the scraper.
func (eng *Engine) run(ctx context.Context) error {
	for {
		url, err := eng.queue.Dequeue(ctx)

		if errors.Is(err, io.EOF) ||
			errors.Is(err, context.Canceled) {
			return nil
		}

		if err := eng.process(ctx, url); err != nil {
			return err
		}
	}
}

// Process processes a single url.
func (eng *Engine) process(ctx context.Context, url *URL) error {
	defer eng.queue.Done(url)

	// Check robots.txt.
	allowed, err := eng.robots.Allowed(ctx, robots.Request{
		URL:       url,
		UserAgent: "ant",
	})
	if err != nil {
		return err
	}
	if !allowed {
		return nil
	}

	// Potential limits.
	if err := eng.limit(ctx, url); err != nil {
		return err
	}

	// Scrape the URL.
	urls, err := eng.scrape(ctx, url)
	if err != nil {
		return err
	}

	// Enqueue URLs.
	if err := eng.enqueue(ctx, urls); err != nil {
		return fmt.Errorf("ant: enqueue - %w", err)
	}

	return nil
}

// Scrape scrapes the given URL and returns the next URLs.
func (eng *Engine) scrape(ctx context.Context, url *URL) (URLs, error) {
	page, err := eng.fetcher.fetch(ctx, url)

	if skip(err) {
		return nil, nil
	}

	if err != nil {
		return nil, fmt.Errorf("ant: fetch %q - %w", url, err)
	}

	defer page.close()

	urls, err := eng.scraper.Scrape(ctx, page)
	if err != nil {
		return nil, fmt.Errorf("ant: scrape %q - %w", url, err)
	}

	return urls, nil
}

// Dedupe de-duplicates the given slice of URLs.
func (eng *Engine) dedupe(ctx context.Context, urls URLs) (URLs, error) {
	deduped, err := eng.deduper.Dedupe(ctx, urls)
	if err != nil {
		return nil, fmt.Errorf("ant: dedupe - %w", err)
	}

	return deduped, nil
}

// Limit runs all configured limiters.
func (eng *Engine) limit(ctx context.Context, url *URL) error {
	if eng.limiter != nil {
		if err := eng.limiter.Limit(ctx, url); err != nil {
			return err
		}
	}

	err := eng.robots.Wait(ctx, robots.Request{
		URL:       url,
		UserAgent: "ant",
	})
	if err != nil {
		return fmt.Errorf("ant: robots wait - %w", err)
	}

	return nil
}

// Matches returns all URLs that match the matcher.
func (eng *Engine) matches(urls URLs) URLs {
	if eng.matcher != nil {
		ret := make(URLs, 0, len(urls))
		for _, u := range urls {
			if eng.matcher.Match(u) {
				ret = append(ret, u)
			}
		}
		return ret
	}
	return urls
}

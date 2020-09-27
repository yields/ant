package main

import (
	"context"
	"log"
	"sync/atomic"
	"time"

	"github.com/yields/ant"
)

type counter struct {
	pages uint64
}

func (c *counter) Scrape(ctx context.Context, p *ant.Page) ([]string, error) {
	atomic.AddUint64(&c.pages, 1)
	return p.URLs(), nil
}

func main() {
	var url = "http://quotes.toscrape.com"
	var ctx = context.Background()
	var scraper = &counter{}
	var start = time.Now()

	eng, err := ant.NewEngine(ant.EngineConfig{
		Scraper: scraper,
		Matcher: ant.MatchHostname("quotes.toscrape.com"),
	})
	if err != nil {
		log.Fatalf("new engine: %s", err)
	}

	if err := eng.Run(ctx, url); err != nil {
		log.Fatal(err)
	}

	log.Printf("scraped %d pages in %s :)",
		scraper.pages,
		time.Since(start),
	)
}

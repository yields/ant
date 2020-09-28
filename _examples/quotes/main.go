package main

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"sync/atomic"
	"time"

	"github.com/yields/ant"
)

type quote struct {
	Text string   `css:".text"`
	By   string   `css:".author"`
	Tags []string `css:".tag"`
}

type scraper struct {
	quotes uint64
	pages  uint64
	enc    *json.Encoder
}

func (s *scraper) Scrape(ctx context.Context, p *ant.Page) (ant.URLs, error) {
	var items struct {
		Quotes []quote `css:".quote"`
	}

	if err := p.Scan(&items); err != nil {
		return nil, err
	}

	for _, q := range items.Quotes {
		if err := s.enc.Encode(q); err != nil {
			return nil, err
		}
	}

	atomic.AddUint64(&s.quotes, uint64(len(items.Quotes)))
	atomic.AddUint64(&s.pages, 1)
	return p.URLs(), nil
}

func main() {
	var url = "http://quotes.toscrape.com"
	var ctx = context.Background()
	var start = time.Now()
	var scraper = &scraper{
		enc:   json.NewEncoder(os.Stdout),
		pages: 0,
	}

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

	log.Printf("scraped %d pages, %d quotes in %s :)",
		scraper.pages,
		scraper.quotes,
		time.Since(start),
	)
}

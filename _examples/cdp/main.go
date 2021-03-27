package main

import (
	"context"
	"log"
	"time"

	"github.com/yields/ant"
	"github.com/yields/ant/antcdp"
)

func main() {
	var apple = "https://apple.com"
	var ctx = context.Background()
	var start = time.Now()

	eng, err := ant.NewEngine(ant.EngineConfig{
		Scraper: title{},
		Fetcher: &ant.Fetcher{
			Client: &antcdp.Client{},
		},
	})
	if err != nil {
		log.Fatalf("new engine: %s", err)
	}

	if err := eng.Run(ctx, apple); err != nil {
		log.Fatalf("run: %s", err)
	}

	log.Printf("done in %s :)", time.Since(start))
}

type title struct{}

func (title) Scrape(ctx context.Context, p *ant.Page) (ant.URLs, error) {
	log.Printf("<title>%s</title>", p.Text("title"))
	return nil, nil
}

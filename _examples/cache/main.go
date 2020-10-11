package main

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/yields/ant"
	"github.com/yields/ant/antcache"
)

type quote struct {
	Text string   `css:".text"   json:"text"`
	By   string   `css:".author" json:"by"`
	Tags []string `css:".tag"    json:"tags"`
}

type page struct {
	Quotes []quote `css:".quote" json:"quotes"`
}

func main() {
	var url = "http://quotes.toscrape.com"
	var ctx = context.Background()
	var start = time.Now()

	disk, err := antcache.Open("/tmp")
	if err != nil {
		log.Printf("open disk: %s", err)
	}

	if err := disk.Wait(ctx); err != nil {
		log.Printf("disk wait: %s", err)
	}

	client, err := antcache.New(
		ant.DefaultClient,
		antcache.Aggressive(24*time.Hour),
		antcache.WithStorage(disk),
	)
	if err != nil {
		log.Printf("new cache: %s", err)
	}

	eng, err := ant.NewEngine(ant.EngineConfig{
		Scraper: ant.JSON(os.Stdout, page{}, `li.next > a`),
		Matcher: ant.MatchHostname("quotes.toscrape.com"),
		Fetcher: &ant.Fetcher{
			Client: client,
		},
	})
	if err != nil {
		log.Fatalf("new engine: %s", err)
	}

	if err := eng.Run(ctx, url); err != nil {
		log.Fatal(err)
	}

	log.Printf("scraped in %s :)", time.Since(start))
}

package main

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/yields/ant"
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

	eng, err := ant.NewEngine(ant.EngineConfig{
		Scraper: ant.JSON(os.Stdout, page{}, `li.next > a`),
		Matcher: ant.MatchHostname("quotes.toscrape.com"),
	})
	if err != nil {
		log.Fatalf("new engine: %s", err)
	}

	if err := eng.Run(ctx, url); err != nil {
		log.Fatal(err)
	}

	log.Printf("scraped in %s :)", time.Since(start))
}

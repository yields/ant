// Package anttest implements scraper test helpers.
//
// Usage:
//
//   func TestScraper(t *testing.T) {
//     var assert = require.New(t)
//     var page = anttest.Fetch(t, "https://github.com")
//     var scraper = &MyScraper{}
//
//     scraper.Scrape(ctx, page)
//
//     assert.Equal("GitHub", scraper.Title)
//   }
//
package anttest

import (
	"context"
	"testing"

	"github.com/yields/ant"
)

// Fetch fetches a page by its URL.
//
// If the page cannot be fetched successfully
// the method calls `t.Fatalf` with the error.
func Fetch(t testing.TB, url string) *ant.Page {
	var ctx = context.Background()

	t.Helper()
	page, err := ant.Fetch(ctx, url)
	if err != nil {
		t.Fatalf("anttest: %s", err)
	}

	return page
}

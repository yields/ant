package ant

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"reflect"
	"sync"
)

// JSON returns a new JSON scraper.
//
// The scraper receives the writer to write JSON lines into
// the type to scrape from pages and optional selectors from
// which to extract the next set of pages to crawl.
//
// The provided type `t` must be a struct, otherwise the scraper
// will return an error on the initial scrape and the crawl engine
// will abort.
//
// The scraper uses the `encoding/json` package to encode the provided
// type into JSON, any errors that are received from the encoder are
// returned from the scraper.
//
// If no selectors are provided, the scraper will return all valid
// URLs on the page.
func JSON(w io.Writer, t interface{}, selectors ...string) Scraper {
	var typ = reflect.TypeOf(t)
	var enc = json.NewEncoder(w)

	if typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}

	return &jsonscraper{
		typ:       typ,
		enc:       enc,
		selectors: selectors,
	}
}

// Jsonscraper implements a json scraper.
type jsonscraper struct {
	typ       reflect.Type
	enc       *json.Encoder
	selectors []string
	lock      sync.Mutex
}

// Scrape implementation.
func (j *jsonscraper) Scrape(ctx context.Context, p *Page) (URLs, error) {
	var v = reflect.New(j.typ)

	if err := p.Scan(v.Interface()); err != nil {
		return nil, err
	}

	if err := j.encode(v.Interface()); err != nil {
		return nil, fmt.Errorf("ant: json encode %s - %w", j.typ, err)
	}

	if len(j.selectors) > 0 {
		var next URLs
		for _, sel := range j.selectors {
			urls, err := p.Next(sel)
			if err != nil {
				return nil, err
			}
			next = append(next, urls...)
		}
		return next, nil
	}

	return p.URLs(), nil
}

// Encode encodes the given v.
func (j *jsonscraper) encode(v interface{}) error {
	j.lock.Lock()
	defer j.lock.Unlock()
	return j.enc.Encode(v)
}

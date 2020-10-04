package ant

import (
	"bytes"
	"context"
	"io"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestJSON(t *testing.T) {
	t.Run("scrapes json", func(t *testing.T) {
		var assert = require.New(t)
		var ctx = context.Background()
		var srv = server(t, "example.com")
		var buf bytes.Buffer
		type data struct {
			Projects []struct {
				Name string `css:"h1"     json:"name"`
			} `css:".project" json:"projects"`
		}

		scraper := JSON(&buf, data{})
		page, err := Fetch(ctx, srv.URL+"/about.html")
		assert.NoError(err)

		_, err = scraper.Scrape(ctx, page)
		assert.NoError(err)

		assert.Equal("{\"projects\":[{\"name\":\"Ant\"}]}\n", buf.String())
	})

	t.Run("scrapes json ptr type", func(t *testing.T) {
		var assert = require.New(t)
		var ctx = context.Background()
		var srv = server(t, "example.com")
		var buf bytes.Buffer
		type data struct {
			Projects []struct {
				Name string `css:"h1"     json:"name"`
			} `css:".project" json:"projects"`
		}

		scraper := JSON(&buf, &data{})
		page, err := Fetch(ctx, srv.URL+"/about.html")
		assert.NoError(err)

		_, err = scraper.Scrape(ctx, page)
		assert.NoError(err)

		assert.Equal("{\"projects\":[{\"name\":\"Ant\"}]}\n", buf.String())
	})

	t.Run("scrape error", func(t *testing.T) {
		var assert = require.New(t)
		var ctx = context.Background()
		var srv = server(t, "example.com")
		var buf bytes.Buffer
		type data struct {
			Projects []struct {
				Name bool `css:"h1"     json:"name"`
			} `css:".project" json:"projects"`
		}

		scraper := JSON(&buf, &data{})
		page, err := Fetch(ctx, srv.URL+"/about.html")
		assert.NoError(err)

		_, err = scraper.Scrape(ctx, page)
		assert.Error(err)
		assert.EqualError(err, `scan: cannot scan into type bool`)
	})

	t.Run("scrape returns all urls", func(t *testing.T) {
		var assert = require.New(t)
		var ctx = context.Background()
		var srv = server(t, "example.com")
		var buf bytes.Buffer
		type data struct {
			Projects []struct {
				Name string `css:"h1"     json:"name"`
			} `css:".project" json:"projects"`
		}

		scraper := JSON(&buf, &data{})
		page, err := Fetch(ctx, srv.URL+"/about.html")
		assert.NoError(err)

		urls, err := scraper.Scrape(ctx, page)
		assert.NoError(err)
		assert.Equal(2, len(urls))
	})

	t.Run("scrape returns URLs matching a selector", func(t *testing.T) {
		var assert = require.New(t)
		var ctx = context.Background()
		var srv = server(t, "example.com")
		var buf bytes.Buffer
		type data struct {
			Projects []struct {
				Name string `css:"h1"     json:"name"`
			} `css:".project" json:"projects"`
		}

		scraper := JSON(&buf, &data{}, `a.next`)
		page, err := Fetch(ctx, srv.URL+"/about.html")
		assert.NoError(err)

		urls, err := scraper.Scrape(ctx, page)
		assert.NoError(err)
		assert.Equal(1, len(urls))
		assert.Equal("/a.html", urls[0].Path)
	})

	t.Run("scrape write error", func(t *testing.T) {
		var assert = require.New(t)
		var ctx = context.Background()
		var srv = server(t, "example.com")
		type data struct {
			Projects []struct {
				Name string `css:"h1"     json:"name"`
			} `css:".project" json:"projects"`
		}

		scraper := JSON(writerError{}, &data{})
		page, err := Fetch(ctx, srv.URL+"/about.html")
		assert.NoError(err)

		_, err = scraper.Scrape(ctx, page)
		assert.Error(err)
		assert.EqualError(err, `ant: json encode ant.data - short write`)
	})
}

type writerError struct{}

func (we writerError) Write(p []byte) (n int, err error) {
	err = io.ErrShortWrite
	return
}

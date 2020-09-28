package ant

import (
	"context"
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDeduper(t *testing.T) {
	t.Run("map", func(t *testing.T) {
		var ctx = context.Background()
		var assert = require.New(t)
		var urls = parseURLs(t, "https://a", "https://b")
		var d = DedupeMap()

		ret, err := d.Dedupe(ctx, urls)
		assert.NoError(err)
		assert.Equal(urls, ret)

		urls = parseURLs(t, "https://a", "https://b", "https://c")
		ret, err = d.Dedupe(ctx, urls)
		assert.NoError(err)
		assert.Equal(urls[2:], ret)
	})

	t.Run("bf", func(t *testing.T) {
		var ctx = context.Background()
		var assert = require.New(t)
		var urls = parseURLs(t, "https://a", "https://b")
		var d = DedupeBF(2000000, 5)

		ret, err := d.Dedupe(ctx, urls)
		assert.NoError(err)
		assert.Equal(urls, ret)

		urls = parseURLs(t, "https://a", "https://b", "https://c")
		ret, err = d.Dedupe(ctx, urls)
		assert.NoError(err)
		assert.Equal(urls[2:], ret)
	})
}

func BenchmarkDedupe(b *testing.B) {
	b.Run("map", func(b *testing.B) {
		var ctx = context.Background()
		var urls = parseURLs(b, "https://a", "https://b")
		var d = DedupeMap()

		for i := 0; i < b.N; i++ {
			d.Dedupe(ctx, urls)
		}
	})

	b.Run("bf", func(b *testing.B) {
		var ctx = context.Background()
		var urls = parseURLs(b, "https://a", "https://b")
		var d = DedupeBF(200000, 5)

		for i := 0; i < b.N; i++ {
			d.Dedupe(ctx, urls)
		}
	})
}

func parseURLs(t testing.TB, rawurls ...string) URLs {
	var ret = make(URLs, 0, len(rawurls))

	t.Helper()

	for _, rawurl := range rawurls {
		u, err := url.Parse(rawurl)
		if err != nil {
			t.Fatalf("parse url %q - %s", rawurl, err)
		}
		ret = append(ret, u)
	}

	return ret
}

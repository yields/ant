package antcache

import (
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestRFC7234(t *testing.T) {
	t.Run("cache", func(t *testing.T) {
		var cases = []struct {
			title string
			req   *http.Request
			cache bool
		}{
			{
				title: "GET",
				req: &http.Request{
					Method: "GET",
				},
				cache: true,
			},
			{
				title: "HEAD",
				req: &http.Request{
					Method: "HEAD",
				},
				cache: true,
			},
			{
				title: "POST",
				req: &http.Request{
					Method: "POST",
				},
				cache: false,
			},
			{
				title: "GET no-store",
				req: &http.Request{
					Method: "GET",
					Header: http.Header{
						"Cache-Control": {"no-store"},
					},
				},
				cache: false,
			},
			{
				title: "GET authorization",
				req: &http.Request{
					Method: "GET",
					Header: http.Header{
						"Authorization": {"token"},
					},
				},
				cache: false,
			},
			{
				title: "GET range",
				req: &http.Request{
					Method: "GET",
					Header: http.Header{
						"Range": {"range"},
					},
				},
				cache: false,
			},
			{
				title: "GET content-range",
				req: &http.Request{
					Method: "GET",
					Header: http.Header{
						"Content-Range": {"range"},
					},
				},
				cache: false,
			},
		}

		for _, c := range cases {
			t.Run(c.title, func(t *testing.T) {
				var assert = require.New(t)
				var strategy = rfc7234{}

				assert.Equal(c.cache, strategy.cache(c.req))
			})
		}
	})

	t.Run("store", func(t *testing.T) {
		var now = time.Now().UTC()
		var cases = []struct {
			title string
			resp  *http.Response
			store bool
		}{
			{
				title: "GET",
				resp: &http.Response{
					StatusCode: 200,
					Header: http.Header{
						"Date":          {now.Format(time.RFC1123)},
						"Cache-Control": {"max-age=5"},
					},
					Request: &http.Request{
						Method: "GET",
					},
				},
				store: true,
			},
			{
				title: "HEAD",
				resp: &http.Response{
					StatusCode: 200,
					Header: http.Header{
						"Date":          {now.Format(time.RFC1123)},
						"Cache-Control": {"max-age=5"},
					},
					Request: &http.Request{
						Method: "HEAD",
					},
				},
				store: true,
			},
			{
				title: "POST",
				resp: &http.Response{
					StatusCode: 200,
					Request: &http.Request{
						Method: "POST",
					},
				},
				store: false,
			},
			{
				title: "GET 500",
				resp: &http.Response{
					StatusCode: 500,
					Request: &http.Request{
						Method: "GET",
					},
				},
				store: false,
			},
			{
				title: "GET request no-store",
				resp: &http.Response{
					StatusCode: 200,
					Request: &http.Request{
						Method: "GET",
						Header: http.Header{
							"Cache-Control": {"no-store"},
						},
					},
				},
				store: false,
			},
			{
				title: "GET response no-store",
				resp: &http.Response{
					StatusCode: 200,
					Header: http.Header{
						"Cache-Control": {"no-store"},
					},
					Request: &http.Request{
						Method: "GET",
					},
				},
				store: false,
			},
			{
				title: "GET response expired",
				resp: &http.Response{
					StatusCode: 200,
					Header: http.Header{
						"Date":    {now.Format(time.RFC1123)},
						"Expires": {now.Add(-time.Minute).Format(time.RFC1123)},
					},
					Request: &http.Request{
						Method: "GET",
					},
				},
				store: false,
			},
			{
				title: "GET response expires",
				resp: &http.Response{
					StatusCode: 200,
					Header: http.Header{
						"Date":    {now.Format(time.RFC1123)},
						"Expires": {now.Add(time.Minute).Format(time.RFC1123)},
					},
					Request: &http.Request{
						Method: "GET",
					},
				},
				store: true,
			},
			{
				title: "GET no explicit cache",
				resp: &http.Response{
					StatusCode: 200,
					Header:     http.Header{},
					Request: &http.Request{
						Method: "GET",
					},
				},
				store: false,
			},
		}

		for _, c := range cases {
			t.Run(c.title, func(t *testing.T) {
				var assert = require.New(t)
				var strategy = rfc7234{}

				assert.Equal(c.store, strategy.store(c.resp))
			})
		}
	})

	t.Run("fresh", func(t *testing.T) {
		var now = time.Now()
		var cases = []struct {
			title string
			resp  *http.Response
			fresh freshness
		}{
			{
				title: "no-cache request",
				resp: &http.Response{
					Request: &http.Request{
						Header: http.Header{
							"Cache-Control": {"no-cache"},
						},
					},
				},
				fresh: stale,
			},
			{
				title: "no-cache response",
				resp: &http.Response{
					Header: http.Header{
						"Cache-Control": {"no-cache"},
					},
					Request: &http.Request{},
				},
				fresh: stale,
			},
			{
				title: "vary mismatch",
				resp: &http.Response{
					Request: &http.Request{
						Header: http.Header{
							"Vary":            {"Accept-Language"},
							"Accept-Language": {"en-US"},
						},
					},
				},
				fresh: transparent,
			},
			{
				title: "stale",
				resp: &http.Response{
					Request: &http.Request{},
				},
				fresh: stale,
			},
			{
				title: "only-if-cached response",
				resp: &http.Response{
					Request: &http.Request{
						Header: http.Header{
							"Cache-Control": {"only-if-cached"},
						},
					},
				},
				fresh: fresh,
			},
			{
				title: "fresh max-age",
				resp: &http.Response{
					Request: &http.Request{},
					Header: http.Header{
						"Date":          {now.Format(time.RFC1123)},
						"Cache-Control": {"max-age=5"},
					},
				},
				fresh: fresh,
			},
			{
				title: "zero max-age",
				resp: &http.Response{
					Request: &http.Request{},
					Header: http.Header{
						"Date":          {now.Format(time.RFC1123)},
						"Cache-Control": {"max-age=0"},
					},
				},
				fresh: stale,
			},
			{
				title: "fresh expires",
				resp: &http.Response{
					Request: &http.Request{},
					Header: http.Header{
						"Date":    {now.Format(time.RFC1123)},
						"Expires": {now.Add(time.Minute).Format(time.RFC1123)},
					},
				},
				fresh: fresh,
			},
			{
				title: "date == expires",
				resp: &http.Response{
					Request: &http.Request{},
					Header: http.Header{
						"Date":    {now.Format(time.RFC1123)},
						"Expires": {now.Format(time.RFC1123)},
					},
				},
				fresh: stale,
			},
			{
				title: "request max-age",
				resp: &http.Response{
					Request: &http.Request{
						Header: http.Header{
							"Cache-Control": {"max-age=5"},
						},
					},
					Header: http.Header{
						"Date": {now.Format(time.RFC1123)},
					},
				},
				fresh: fresh,
			},
			{
				title: "request zero max-age",
				resp: &http.Response{
					Request: &http.Request{
						Header: http.Header{
							"Cache-Control": {"max-age=0"},
						},
					},
					Header: http.Header{
						"Date": {now.Format(time.RFC1123)},
					},
				},
				fresh: stale,
			},
			{
				title: "request min-fresh",
				resp: &http.Response{
					Request: &http.Request{
						Header: http.Header{
							"Cache-Control": {"min-fresh=3"},
						},
					},
					Header: http.Header{
						"Date":          {now.Format(time.RFC1123)},
						"Cache-Control": {"max-age=5"},
					},
				},
				fresh: fresh,
			},
			{
				title: "request min-fresh reached",
				resp: &http.Response{
					Request: &http.Request{
						Header: http.Header{
							"Cache-Control": {"min-fresh=5"},
						},
					},
					Header: http.Header{
						"Date":          {now.Format(time.RFC1123)},
						"Cache-Control": {"max-age=5"},
					},
				},
				fresh: stale,
			},
			{
				title: "max-stale",
				resp: &http.Response{
					Request: &http.Request{
						Header: http.Header{
							"Cache-Control": {"max-stale"},
						},
					},
					Header: http.Header{
						"Date": {now.Format(time.RFC1123)},
					},
				},
				fresh: fresh,
			},
		}

		for _, c := range cases {
			t.Run(c.title, func(t *testing.T) {
				var assert = require.New(t)
				var strategy = rfc7234{}

				expect := c.fresh
				got := strategy.fresh(c.resp)

				assert.Equal(expect.String(), got.String())
			})
		}
	})
}

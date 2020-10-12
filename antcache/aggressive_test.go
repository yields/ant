package antcache

import (
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestAggressive(t *testing.T) {
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
				var strategy = aggressive{}

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
						"Date": {now.Format(time.RFC1123)},
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
						"Date": {now.Format(time.RFC1123)},
					},
					Request: &http.Request{
						Method: "HEAD",
					},
				},
				store: true,
			},
			{
				title: "no date header",
				resp: &http.Response{
					StatusCode: 200,
					Request: &http.Request{
						Method: "HEAD",
					},
				},
				store: false,
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
		}

		for _, c := range cases {
			t.Run(c.title, func(t *testing.T) {
				var assert = require.New(t)
				var strategy = aggressive{}

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
				title: "fresh",
				resp: &http.Response{
					Request: &http.Request{},
					Header: http.Header{
						"Date": {now.Format(time.RFC1123)},
					},
				},
				fresh: fresh,
			},
			{
				title: "fresh 2 hours",
				resp: &http.Response{
					Request: &http.Request{},
					Header: http.Header{
						"Date": {now.Add(-(2 * time.Hour)).Format(time.RFC1123)},
					},
				},
				fresh: fresh,
			},
			{
				title: "transparent 2 days",
				resp: &http.Response{
					Request: &http.Request{},
					Header: http.Header{
						"Date": {now.Add(-(48 * time.Hour)).Format(time.RFC1123)},
					},
				},
				fresh: transparent,
			},
			{
				title: "transparent",
				resp: &http.Response{
					Request: &http.Request{},
				},
				fresh: transparent,
			},
		}

		for _, c := range cases {
			t.Run(c.title, func(t *testing.T) {
				var assert = require.New(t)
				var strategy = aggressive{}

				assert.Equal(c.fresh, strategy.fresh(c.resp))
			})
		}
	})
}

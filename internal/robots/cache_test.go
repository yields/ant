package robots

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCache(t *testing.T) {
	t.Run("allowed", func(t *testing.T) {
		var ctx = context.Background()
		var assert = require.New(t)
		var cache = NewCache(http.DefaultClient, 50)
		var url = serve(t, "testdata/robots.txt")

		req := request(t, url+"/foo", "ant")

		allowed, err := cache.Allowed(ctx, req)
		assert.NoError(err)
		assert.True(allowed)
	})

	t.Run("allowed cancel", func(t *testing.T) {
		var ctx = context.Background()
		var assert = require.New(t)
		var cache = NewCache(http.DefaultClient, 50)
		var url = serve(t, "testdata/robots.txt")

		ctx, cancel := context.WithCancel(ctx)
		cancel()

		req := request(t, url+"/foo", "ant")

		_, err := cache.Allowed(ctx, req)
		assert.Error(err)
		assert.True(errors.Is(err, context.Canceled))
	})

	t.Run("disallow", func(t *testing.T) {
		var ctx = context.Background()
		var assert = require.New(t)
		var cache = NewCache(http.DefaultClient, 50)
		var url = serve(t, "testdata/robots.txt")

		req := request(t, url+"/search", "ant")

		allowed, err := cache.Allowed(ctx, req)
		assert.NoError(err)
		assert.False(allowed)
	})

	t.Run("delay", func(t *testing.T) {
		var ctx = context.Background()
		var assert = require.New(t)
		var cache = NewCache(http.DefaultClient, 50)
		var url = serve(t, "testdata/robots.txt")

		req := request(t, url, "badbot")

		err := cache.Wait(ctx, req)
		assert.NoError(err)
	})

	t.Run("delay cancel", func(t *testing.T) {
		var ctx = context.Background()
		var assert = require.New(t)
		var cache = NewCache(http.DefaultClient, 50)
		var url = serve(t, "testdata/robots.txt")

		ctx, cancel := context.WithCancel(ctx)
		cancel()

		req := request(t, url, "badbot")

		err := cache.Wait(ctx, req)
		assert.Error(err)
		assert.True(errors.Is(err, context.Canceled))
	})
}

func BenchmarkCache(b *testing.B) {
	b.Run("allowed", func(b *testing.B) {
		var ctx = context.Background()
		var cache = NewCache(http.DefaultClient, 50)
		var url = serve(b, "testdata/robots.txt")
		var req = request(b, url+"/foo", "ant")

		for i := 0; i < b.N; i++ {
			if _, err := cache.Allowed(ctx, req); err != nil {
				b.Fatalf("allowed: %s", err)
			}
		}
	})
}

func request(t testing.TB, rawurl, ua string) Request {
	t.Helper()

	u, err := url.Parse(rawurl)
	if err != nil {
		t.Fatalf("parse: %s", err)
	}

	return Request{
		URL:       u,
		UserAgent: ua,
	}
}

func serve(t testing.TB, path string) (uri string) {
	t.Helper()

	serve := func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/robots.txt" {
			http.ServeFile(w, r, path)
		}
	}

	srv := httptest.NewServer(http.HandlerFunc(serve))

	t.Cleanup(func() {
		srv.Close()
	})

	return srv.URL
}

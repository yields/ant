package ant

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestFetcher(t *testing.T) {
	minBackoff = time.Nanosecond
	maxBackoff = time.Millisecond

	t.Run("fetch bad URL", func(t *testing.T) {
		var assert = require.New(t)
		var ctx = context.Background()

		_, err := Fetch(ctx, "")

		assert.Error(err)
		assert.Contains(err.Error(), `ant: GET "" - Get "": unsupported protocol scheme ""`)
	})

	t.Run("simple", func(t *testing.T) {
		var assert = require.New(t)
		var fetcher = &Fetcher{}
		var srv = server(t, "example.com")
		var ctx = context.Background()
		var u = parseURL(t, srv.URL)

		p, err := fetcher.Fetch(ctx, u)

		assert.NoError(err)
		assert.Equal("Example", p.Text("title"))
	})

	t.Run("400", func(t *testing.T) {
		var ctx = context.Background()
		var assert = require.New(t)
		var fetcher = &Fetcher{}
		var url = serve(t, respond(400, ""))

		_, err := fetcher.Fetch(ctx, url)

		assert.Error(err)
		assert.Contains(err.Error(), `400 Bad Request`)
	})

	t.Run("404", func(t *testing.T) {
		var ctx = context.Background()
		var assert = require.New(t)
		var fetcher = &Fetcher{}
		var url = serve(t, respond(404, ""))

		p, err := fetcher.Fetch(ctx, url)

		assert.NoError(err)
		assert.Nil(p)
	})

	t.Run("fetch error", func(t *testing.T) {
		var ctx = context.Background()
		var assert = require.New(t)
		var fetcher = &Fetcher{}
		var url = serve(t, respond(400, ""))

		_, err := fetcher.Fetch(ctx, url)
		assert.Error(err)

		e, ok := err.(*FetchError)
		assert.True(ok, "expected a fetch error")
		assert.Equal(400, e.Status)
	})

	t.Run("fetch retry", func(t *testing.T) {
		var ctx = context.Background()
		var assert = require.New(t)
		var fetcher = &Fetcher{}
		var reqs uint64
		var url = serve(t, func(w http.ResponseWriter) {
			if atomic.AddUint64(&reqs, 1) == 3 {
				w.WriteHeader(200)
				return
			}
			w.WriteHeader(503)
		})

		p, err := fetcher.Fetch(ctx, url)
		assert.NoError(err)
		assert.NotNil(p)
		assert.NoError(p.close())
		assert.Equal(uint64(3), reqs)
	})

	t.Run("fetch max attempts reached", func(t *testing.T) {
		var ctx = context.Background()
		var assert = require.New(t)
		var fetcher = &Fetcher{}
		var url = serve(t, func(w http.ResponseWriter) {
			w.WriteHeader(503)
		})

		_, err := fetcher.Fetch(ctx, url)
		assert.Error(err)
		assert.Contains(err.Error(), "max attempts of 5 reached")
		assert.Contains(err.Error(), `503`)
	})

	t.Run("sends headers", func(t *testing.T) {
		var ctx = context.Background()
		var assert = require.New(t)
		var fetcher = &Fetcher{}
		var req http.Request
		var url = record(t, &req)

		_, err := fetcher.Fetch(ctx, url)
		assert.NoError(err)

		assert.Equal("text/html; charset=UTF-8", req.Header.Get("Accept"))
		assert.Equal(UserAgent.String(), req.Header.Get("User-Agent"))
	})

	t.Run("custom user-agent", func(t *testing.T) {
		var ctx = context.Background()
		var assert = require.New(t)
		var fetcher = &Fetcher{}
		var req http.Request
		var url = record(t, &req)

		fetcher.UserAgent = StaticAgent("foo")
		_, err := fetcher.Fetch(ctx, url)
		assert.NoError(err)

		assert.Equal("text/html; charset=UTF-8", req.Header.Get("Accept"))
		assert.Equal("foo", req.Header.Get("User-Agent"))
	})
}

func respond(status int, body string) func(http.ResponseWriter) {
	return func(w http.ResponseWriter) {
		w.WriteHeader(status)
		io.WriteString(w, body)
	}
}

func serve(t testing.TB, f func(w http.ResponseWriter)) *URL {
	t.Helper()

	serve := func(w http.ResponseWriter, r *http.Request) {
		f(w)
	}

	srv := httptest.NewServer(http.HandlerFunc(serve))
	t.Cleanup(func() {
		srv.Close()
	})

	return parseURL(t, srv.URL)
}

func record(t testing.TB, req *http.Request) *URL {
	t.Helper()

	serve := func(w http.ResponseWriter, r *http.Request) {
		*req = *r.Clone(context.Background())
		w.WriteHeader(200)
	}

	srv := httptest.NewServer(http.HandlerFunc(serve))
	t.Cleanup(func() {
		srv.Close()
	})

	return parseURL(t, srv.URL)
}

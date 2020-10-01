package ant

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFetcher(t *testing.T) {
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

		p, err := fetcher.fetch(ctx, u)

		assert.NoError(err)
		assert.Equal("Example", p.Text("title"))
	})

	t.Run("4xx", func(t *testing.T) {
		var ctx = context.Background()
		var assert = require.New(t)
		var fetcher = &Fetcher{}
		var url = serve(t, 400, "")

		_, err := fetcher.fetch(ctx, url)

		assert.Error(err)
		assert.Contains(err.Error(), `400 Bad Request`)
	})

	t.Run("fetch error", func(t *testing.T) {
		var ctx = context.Background()
		var assert = require.New(t)
		var fetcher = &Fetcher{}
		var url = serve(t, 400, "")

		_, err := fetcher.fetch(ctx, url)
		assert.Error(err)

		e, ok := err.(*FetchError)
		assert.True(ok, "expected a fetch error")
		assert.Equal(400, e.Status)
	})

	t.Run("sends headers", func(t *testing.T) {
		var ctx = context.Background()
		var assert = require.New(t)
		var fetcher = &Fetcher{}
		var req http.Request
		var url = record(t, &req)

		_, err := fetcher.fetch(ctx, url)
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
		_, err := fetcher.fetch(ctx, url)
		assert.NoError(err)

		assert.Equal("text/html; charset=UTF-8", req.Header.Get("Accept"))
		assert.Equal("foo", req.Header.Get("User-Agent"))
	})
}

func serve(t testing.TB, status int, body string) *URL {
	t.Helper()

	serve := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(status)
		io.WriteString(w, body)
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

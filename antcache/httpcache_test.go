package antcache

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestCache(t *testing.T) {
	t.Run("freshness", func(t *testing.T) {
		var cases = []struct {
			freshness freshness
			output    string
		}{
			{fresh, "fresh"},
			{stale, "stale"},
			{transparent, "transprent"},
			{-1, "antcache.freshness(-1)"},
		}

		for _, c := range cases {
			t.Run(c.output, func(t *testing.T) {
				var assert = require.New(t)

				actual := c.freshness.String()
				expect := c.output

				assert.Equal(expect, actual)
			})
		}
	})

	t.Run("new nil client", func(t *testing.T) {
		var assert = require.New(t)

		_, err := New(nil)

		assert.EqualError(err, `antcache: client must be non-nil`)
	})

	t.Run("new nil storage", func(t *testing.T) {
		var assert = require.New(t)

		_, err := New(http.DefaultClient, WithStorage(nil))

		assert.EqualError(err, `antcache: storage must be non-nil`)
	})

	t.Run("defers to client if request is not cacheable", func(t *testing.T) {
		var assert = require.New(t)
		var srv = server(t)
		var req = request(t, srv.url)

		req.Header.Set("Cache-Control", "no-store")

		c, err := New(http.DefaultClient)
		assert.NoError(err)

		resp, err := c.Do(req)
		assert.NoError(err)
		assert.Equal(200, resp.StatusCode)
		assert.Equal(uint64(1), srv.requests())

		read(t, resp)
	})

	t.Run("caches responses", func(t *testing.T) {
		var assert = require.New(t)
		var srv = server(t)
		var req = request(t, srv.url)

		c, err := New(http.DefaultClient)
		assert.NoError(err)

		resp, err := c.Do(req)
		assert.NoError(err)
		assert.Equal(200, resp.StatusCode)
		assert.Equal(uint64(1), srv.requests())

		read(t, resp)

		resp, err = c.Do(req)
		assert.NoError(err)
		assert.Equal(200, resp.StatusCode)
		assert.Equal("1", resp.Header.Get("X-From-Cache"))
		assert.Equal(uint64(1), srv.requests())
	})

	t.Run("verifies a cached response", func(t *testing.T) {
		var assert = require.New(t)
		var srv = server(t)
		var req = request(t, srv.url)

		c, err := New(http.DefaultClient)
		assert.NoError(err)

		resp, err := c.Do(req)
		assert.NoError(err)
		assert.Equal(200, resp.StatusCode)
		assert.Equal(uint64(1), srv.requests())

		read(t, resp)

		req.Header.Set("Cache-Control", "max-age=0")

		resp, err = c.Do(req)
		assert.NoError(err)
		assert.Equal("1", resp.Header.Get("X-Verified"))
	})

	t.Run("re-uses a cached response when stale-if-error", func(t *testing.T) {
		var assert = require.New(t)
		var srv = server(t)
		var req = request(t, srv.url)

		c, err := New(http.DefaultClient)
		assert.NoError(err)

		resp, err := c.Do(req)
		assert.NoError(err)
		assert.Equal(200, resp.StatusCode)
		assert.Equal(uint64(1), srv.requests())

		read(t, resp)

		req.Header.Set("Cache-Control", "max-age=0, stale-if-error")
		req.Header.Set("X-Status", "500")

		newresp, err := c.Do(req)
		assert.NoError(err)
		assert.Equal(200, newresp.StatusCode)
		assert.Equal(resp.Header.Get("Date"), newresp.Header.Get("Date"))
	})

	t.Run("returns verified response when stale-if-error is not set", func(t *testing.T) {
		var assert = require.New(t)
		var srv = server(t)
		var req = request(t, srv.url)

		c, err := New(http.DefaultClient)
		assert.NoError(err)

		resp, err := c.Do(req)
		assert.NoError(err)
		assert.Equal(200, resp.StatusCode)
		assert.Equal(uint64(1), srv.requests())

		read(t, resp)

		req.Header.Set("Cache-Control", "max-age=0")
		req.Header.Set("X-Status", "500")

		newresp, err := c.Do(req)
		assert.NoError(err)
		assert.Equal(500, newresp.StatusCode)
	})

	t.Run("stores a new response when etag does not match", func(t *testing.T) {
		var assert = require.New(t)
		var srv = server(t)
		var req = request(t, srv.url)

		c, err := New(http.DefaultClient)
		assert.NoError(err)

		resp, err := c.Do(req)
		assert.NoError(err)
		assert.Equal(200, resp.StatusCode)
		assert.Equal(uint64(1), srv.requests())

		read(t, resp)

		req.Header.Set("If-None-Match", "foo")
		req.Header.Set("Cache-Control", "max-age=0")

		newresp, err := c.Do(req)
		assert.NoError(err)
		assert.Equal("", newresp.Header.Get("X-Verified"))
	})
}

type serverInfo struct {
	url string
	req uint64
}

func (si *serverInfo) requests() uint64 {
	return atomic.LoadUint64(&si.req)
}

func server(t testing.TB) *serverInfo {
	t.Helper()

	info := &serverInfo{}
	body := `
		<!DOCTYPE html>
		<html>
		  <head>
		    <title>Example</title>
		  </head>
		  <body>
		    <a href="/about.html"></a>
		    <a href="/products.html"></a>
		    <a href="/search.html"></a>
		  </body>
		</html>
	`

	serve := func(w http.ResponseWriter, r *http.Request) {
		var now = time.Now().Format(time.RFC1123)

		atomic.AddUint64(&info.req, 1)

		w.Header().Set("Date", now)
		w.Header().Set("Cache-Control", "max-age=120")
		w.Header().Set("ETag", "etag")
		w.Header().Set("Last-Modified", now)

		if status, _ := strconv.Atoi(r.Header.Get("X-Status")); status != 0 {
			w.WriteHeader(status)
			return
		}

		if etag := r.Header.Get("If-None-Match"); etag == "etag" {
			w.Header().Set("X-Verified", "1")
			w.WriteHeader(304)
			return
		}

		w.WriteHeader(200)
		w.Write([]byte(body))
	}

	srv := httptest.NewServer(http.HandlerFunc(serve))
	info.url = srv.URL

	t.Cleanup(func() {
		srv.Close()
	})

	return info
}

func request(t testing.TB, url string) *http.Request {
	t.Helper()

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		t.Fatalf("new request: %s", err)
	}

	return req
}

func read(t testing.TB, resp *http.Response) {
	t.Helper()
	io.Copy(io.Discard, resp.Body)
	if err := resp.Body.Close(); err != nil {
		t.Fatalf("read: %s", err)
	}
}

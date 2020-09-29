package antcdp

import (
	"io/ioutil"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestClient(t *testing.T) {
	t.Run("performs a request", func(t *testing.T) {
		var assert = require.New(t)
		var srv = serve(t, "testdata/simple.html")
		var req = request(t, srv.URL)
		var client = &Client{}

		resp, err := client.Do(req)

		assert.NoError(err)

		buf, err := ioutil.ReadAll(resp.Body)
		assert.NoError(err)
		assert.Equal(200, resp.StatusCode)
		assert.Contains(string(buf), `<title>simple</title>`)
	})

	t.Run("reads response", func(t *testing.T) {
		var assert = require.New(t)
		var srv = serve(t, "testdata/404.html")
		var req = request(t, srv.URL)
		var client = &Client{}

		resp, err := client.Do(req)
		assert.NoError(err)

		expect := &http.Response{
			Status:        "404 Not Found",
			StatusCode:    404,
			Proto:         "http/1.1",
			ProtoMajor:    1,
			ProtoMinor:    1,
			Header:        resp.Header,
			Body:          resp.Body,
			ContentLength: 123,
			Uncompressed:  true,
			Request:       resp.Request,
		}

		assert.Equal(expect, resp)
	})

	t.Run("adjusts content length header", func(t *testing.T) {
		var assert = require.New(t)
		var srv = serve(t, "testdata/404.html")
		var req = request(t, srv.URL)
		var client = &Client{}

		resp, err := client.Do(req)
		assert.NoError(err)
		assert.Equal(int64(123), resp.ContentLength)
		assert.Equal("123", resp.Header.Get("Content-Length"))
	})
}

func request(t testing.TB, rawurl string) *http.Request {
	t.Helper()

	u, err := url.Parse(rawurl)
	if err != nil {
		t.Fatalf("parse url: %s", err)
	}

	_, port, err := net.SplitHostPort(u.Host)
	if err != nil {
		t.Fatalf("split hostport: %s", err)
	}

	if runtime.GOOS == "darwin" {
		u.Host = "host.docker.internal:" + port
	}

	return &http.Request{
		Method: "GET",
		Header: make(http.Header),
		URL:    u,
	}
}

func serve(t testing.TB, path string) *httptest.Server {
	t.Helper()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, path)
	}))

	t.Cleanup(func() {
		srv.Close()
	})

	return srv
}

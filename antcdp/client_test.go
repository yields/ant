package antcdp

import (
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"os/exec"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestClient(t *testing.T) {
	if os.Getenv("HEADLESS_SHELL") == "" {
		t.Skipf("skipped antcdp tests, set HEADLESS_SHELL to run.")
	}

	boot(t, os.Getenv("HEADLESS_DEBUG") == "1")

	t.Run("performs a request", func(t *testing.T) {
		var assert = require.New(t)
		var srv = serve(t, "testdata/simple.html")
		var req = request(t, srv.URL)
		var client = setup(t)

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
		var client = setup(t)

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
		var client = setup(t)

		resp, err := client.Do(req)
		assert.NoError(err)
		assert.Equal(int64(123), resp.ContentLength)
		assert.Equal("123", resp.Header.Get("Content-Length"))
	})

	t.Run("sets and reads cookies", func(t *testing.T) {
		var assert = require.New(t)
		var srv = serve(t, "testdata/cookies.html")
		var req = request(t, srv.URL)
		var client = setup(t)

		req.AddCookie(&http.Cookie{
			Name:  "key",
			Value: "value",
		})

		resp, err := client.Do(req)
		assert.NoError(err)
		assert.Equal(200, resp.StatusCode)
		assert.Equal(2, len(resp.Cookies()))

		assert.Equal("key", resp.Cookies()[0].Name)
		assert.Equal("value", resp.Cookies()[0].Value)

		assert.Equal("js_cookie", resp.Cookies()[1].Name)
		assert.Equal("true", resp.Cookies()[1].Value)
	})
}

var (
	bin  = os.Getenv("HEADLESS_SHELL")
	args = [...]string{
		"--no-sandbox",
		"--remote-debugging-address=127.0.0.1",
		"--remote-debugging-port=9222",
	}
)

func boot(t testing.TB, debug bool) {
	var cmd = exec.Command(bin, args[:]...)
	var errc = make(chan error)
	var addr = "127.0.0.1:9222"
	var maxattempts = 5
	var backoff = 1 * time.Second
	var ready bool

	if debug {
		t.Logf("$ %s %v ", bin, args)
		cmd.Stderr = os.Stderr
		cmd.Stdout = os.Stdout
	}

	if err := cmd.Start(); err != nil {
		log.Fatalf("$ %s %v => %s", bin, args, err)
	}

	for j := 0; j < maxattempts; j++ {
		c, err := net.Dial("tcp", addr)
		if err != nil {
			time.Sleep(backoff)
			continue
		}
		c.Close()
		ready = true
	}

	if !ready {
		t.Fatalf("cannot connect to headless-shell at %s", addr)
	}

	go func() {
		errc <- cmd.Wait()
	}()

	t.Cleanup(func() {
		if debug {
			t.Logf("cleanup")
		}
		if err := cmd.Process.Kill(); err != nil {
			t.Fatalf("kill %s - %s", bin, err)
		}
		<-errc
	})
}

func setup(t *testing.T) *Client {
	t.Helper()
	t.Parallel()

	return &Client{}
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

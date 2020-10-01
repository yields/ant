package ant

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"path"
	"sort"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEngine(t *testing.T) {
	t.Run("nil scraper", func(t *testing.T) {
		var assert = require.New(t)

		_, err := NewEngine(EngineConfig{})

		assert.Error(err)
		assert.EqualError(err, `ant: scraper is required`)
	})

	t.Run("run", func(t *testing.T) {
		var ctx = context.Background()
		var assert = require.New(t)
		var visitor = &visitor{}
		var eng = setup(t, visitor)
		var srv = server(t, "example.com")

		err := eng.Run(ctx, srv.URL)

		assert.NoError(err)

		sort.Strings(visitor.paths)
		expect := []string{
			"/",
			"/a.html",
			"/about.html",
			"/b.html",
			"/products.html",
		}

		assert.Equal(expect, visitor.paths)
	})

	t.Run("run with matcher", func(t *testing.T) {
		var ctx = context.Background()
		var assert = require.New(t)
		var visitor = &visitor{}
		var eng = setup(t, visitor)
		var srv = server(t, "example.com")

		eng.matcher = MatchPattern("*/")
		err := eng.Run(ctx, srv.URL)

		assert.NoError(err)

		expect := []string{"/"}
		assert.Equal(expect, visitor.paths)
	})

	t.Run("run aborts when a scraper errors", func(t *testing.T) {
		var ctx = context.Background()
		var assert = require.New(t)
		var scraper = &scraperError{n: 2, err: io.ErrUnexpectedEOF}
		var eng = setup(t, scraper)
		var srv = server(t, "example.com")

		err := eng.Run(ctx, srv.URL)
		assert.Error(err)
		assert.Contains(err.Error(), `unexpected EOF`)
	})

	t.Run("cancel", func(t *testing.T) {
		var ctx = context.Background()
		var assert = require.New(t)
		var visitor = &visitor{}
		var eng = setup(t, visitor)
		var srv = server(t, "example.com")

		subctx, cancel := context.WithCancel(ctx)
		cancel()

		err := eng.Run(subctx, srv.URL+"?wait=1s")

		assert.Error(err)
		assert.True(errors.Is(err, context.Canceled), err.Error())
	})

	t.Run("enqueue invalid URL", func(t *testing.T) {
		var ctx = context.Background()
		var assert = require.New(t)
		var eng = setup(t, &visitor{})

		err := eng.Enqueue(ctx, "foo")

		assert.Error(err)
		assert.EqualError(err, `ant: cannot enqueue invalid URL "foo"`)
	})

	t.Run("fetch error", func(t *testing.T) {
		var ctx = context.Background()
		var assert = require.New(t)
		var eng = setup(t, &visitor{})

		eng.impolite = true
		err := eng.Run(ctx, "http://:9999")

		assert.Error(err)
		assert.Contains(err.Error(), `connection refused`)
	})
}

func BenchmarkEngine(b *testing.B) {
	b.Run("enqueue", func(b *testing.B) {
		var ctx = context.Background()
		var visitor = &visitor{}
		var eng = setup(b, visitor)
		var srv = server(b, "example.com")
		defer srv.Close()

		b.ResetTimer()

		for j := 0; j < b.N; j++ {
			eng.Enqueue(ctx, srv.URL)
		}
	})
}

// Visitor implements a scraper
// that collects all visited paths.
type visitor struct {
	paths []string
	mtx   sync.Mutex
}

// Scrape implementation.
func (v *visitor) Scrape(ctx context.Context, p *Page) (URLs, error) {
	v.mtx.Lock()
	v.paths = append(v.paths, p.URL.Path)
	v.mtx.Unlock()
	return p.URLs(), nil
}

// ScraperError returns an error after
// N calls to scrape.
type scraperError struct {
	n   int
	seq uint64
	err error
}

// Scrape implementation.
func (s *scraperError) Scrape(ctx context.Context, p *Page) (URLs, error) {
	if atomic.AddUint64(&s.seq, 1) == uint64(s.n) {
		return nil, s.err
	}
	return p.URLs(), nil
}

// Setup a new engine using a scraper.
func setup(t testing.TB, s Scraper) *Engine {
	t.Helper()

	eng, err := NewEngine(EngineConfig{
		Concurrency: 1,
		Scraper:     s,
	})
	if err != nil {
		t.Fatalf("new engine: %s", err)
	}

	return eng
}

func server(t testing.TB, dir string) *httptest.Server {
	t.Helper()

	dir = path.Join("testdata", dir)
	fs := http.FileServer(http.Dir(dir))

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fs.ServeHTTP(w, r)
	}))

	t.Cleanup(func() {
		srv.Close()
	})

	return srv
}

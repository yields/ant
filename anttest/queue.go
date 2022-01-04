package anttest

import (
	"context"
	"io"
	"net/url"
	"sort"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/yields/ant"
	"golang.org/x/sync/errgroup"
)

// TestQueue tests a Queue implementation.
//
// `new(t)` must return a new empty queue ready for use.
func TestQueue(t *testing.T, new func(testing.TB) ant.Queue) {
	t.Run("enqueue dequeue", func(t *testing.T) {
		var ctx = context.Background()
		var assert = require.New(t)
		var urls = parseURLs(t, "https://a", "https://b")
		var queue = new(t)

		err := queue.Enqueue(ctx, urls)
		assert.NoError(err)

		a, err := queue.Dequeue(ctx)
		assert.NoError(err)
		assert.Equal("https://a", a.String())

		b, err := queue.Dequeue(ctx)
		assert.NoError(err)
		assert.Equal("https://b", b.String())
	})

	t.Run("enqueue multi", func(t *testing.T) {
		var ctx = context.Background()
		var assert = require.New(t)
		var queue = new(t)

		for j := 0; j < 1000; j++ {
			urls := parseURLs(t, "https://"+strconv.Itoa(j))
			err := queue.Enqueue(ctx, urls)
			assert.NoError(err)
		}
	})

	t.Run("enqueue canceled context", func(t *testing.T) {
		var ctx = context.Background()
		var assert = require.New(t)
		var urls = parseURLs(t, "https://a", "https://b")
		var queue = new(t)

		ctx, cancel := context.WithCancel(ctx)
		cancel()

		err := queue.Enqueue(ctx, urls)
		assert.Equal(context.Canceled, err)
	})

	t.Run("enqueue closed", func(t *testing.T) {
		var ctx = context.Background()
		var assert = require.New(t)
		var urls = parseURLs(t, "https://a", "https://b")
		var queue = new(t)

		err := queue.Close(ctx)
		assert.NoError(err)

		err = queue.Enqueue(ctx, urls)
		assert.Equal(io.EOF, err)
	})

	t.Run("dequeue multi readers", func(t *testing.T) {
		var ctx = context.Background()
		var assert = require.New(t)
		var queue = new(t)
		var recv = make([]string, 3)

		eg, subctx := errgroup.WithContext(ctx)

		for j := 0; j < 3; j++ {
			j := j
			eg.Go(func() error {
				u, err := queue.Dequeue(subctx)
				if err != nil {
					return err
				}
				recv[j] = u.String()
				return err
			})
		}

		urls := parseURLs(t, "https://a", "https://b", "https://c")
		err := queue.Enqueue(ctx, urls)
		assert.NoError(err)

		err = eg.Wait()
		assert.NoError(err)

		sort.Strings(recv)

		assert.Equal(recv, urlStrings(t, urls))
	})

	t.Run("dequeue closed", func(t *testing.T) {
		var ctx = context.Background()
		var assert = require.New(t)
		var queue = new(t)

		err := queue.Close(ctx)
		assert.NoError(err)

		_, err = queue.Dequeue(ctx)
		assert.Equal(io.EOF, err)
	})

	t.Run("dequeue canceled context", func(t *testing.T) {
		var ctx = context.Background()
		var assert = require.New(t)
		var queue = new(t)

		ctx, cancel := context.WithCancel(ctx)
		cancel()

		_, err := queue.Dequeue(ctx)
		assert.Equal(context.Canceled, err)
	})
}

// BenchmarkQueue benchmarks a queue implementation.
func BenchmarkQueue(b *testing.B, new func(testing.TB) ant.Queue) {
	var ctx = context.Background()
	var urls = parseURLs(b, "https://a")
	var queue = new(b)

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			if err := queue.Enqueue(ctx, urls); err != nil {
				b.Fatalf("enqueue: %s", err)
			}
			if _, err := queue.Dequeue(ctx); err != nil {
				b.Fatalf("dequeue: %s", err)
			}
		}
	})

	queue.Close(ctx)
}

func parseURLs(t testing.TB, rawurls ...string) []*url.URL {
	var ret = make([]*url.URL, 0, len(rawurls))

	t.Helper()

	for _, rawurl := range rawurls {
		u, err := url.Parse(rawurl)
		if err != nil {
			t.Fatalf("anttest: parse url %q - %s", rawurl, err)
		}
		ret = append(ret, u)
	}

	return ret
}

func urlStrings(t testing.TB, urls []*url.URL) []string {
	var ret = make([]string, 0, len(urls))

	t.Helper()

	for _, u := range urls {
		ret = append(ret, u.String())
	}

	sort.Strings(ret)
	return ret
}

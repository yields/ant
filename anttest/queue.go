package anttest

import (
	"context"
	"io"
	"sort"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/yields/ant"
	"golang.org/x/sync/errgroup"
)

// Queue tests a Queue implementation.
//
// `new(t)` must return a new empty queue ready for use.
func Queue(t *testing.T, new func(testing.TB) ant.Queue) {
	t.Run("enqueue dequeue", func(t *testing.T) {
		var ctx = context.Background()
		var assert = require.New(t)
		var queue = new(t)

		err := queue.Enqueue(ctx, "a", "b")
		assert.NoError(err)

		a, err := queue.Dequeue(ctx)
		assert.NoError(err)
		assert.Equal("a", a)

		b, err := queue.Dequeue(ctx)
		assert.NoError(err)
		assert.Equal("b", b)
	})

	t.Run("enqueue multi", func(t *testing.T) {
		var ctx = context.Background()
		var assert = require.New(t)
		var queue = new(t)

		for j := 0; j < 1000; j++ {
			err := queue.Enqueue(ctx, strconv.Itoa(j))
			assert.NoError(err)
		}
	})

	t.Run("enqueue canceled context", func(t *testing.T) {
		var ctx = context.Background()
		var assert = require.New(t)
		var queue = new(t)

		ctx, cancel := context.WithCancel(ctx)
		cancel()

		err := queue.Enqueue(ctx, "a", "b")
		assert.Equal(context.Canceled, err)
	})

	t.Run("enqueue closed", func(t *testing.T) {
		var ctx = context.Background()
		var assert = require.New(t)
		var queue = new(t)

		err := queue.Close()
		assert.NoError(err)

		err = queue.Enqueue(ctx, "a", "b")
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
				recv[j] = u
				return err
			})
		}

		err := queue.Enqueue(ctx, "a", "b", "c")
		assert.NoError(err)

		err = eg.Wait()
		assert.NoError(err)

		sort.Strings(recv)
		assert.Equal([]string{"a", "b", "c"}, recv)
	})

	t.Run("dequeue closed", func(t *testing.T) {
		var ctx = context.Background()
		var assert = require.New(t)
		var queue = new(t)

		err := queue.Close()
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

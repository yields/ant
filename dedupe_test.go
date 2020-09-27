package ant

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDeduper(t *testing.T) {
	t.Run("map", func(t *testing.T) {
		var ctx = context.Background()
		var assert = require.New(t)
		var d = DedupeMap()

		ret, err := d.Dedupe(ctx, []string{"a", "b"})
		assert.NoError(err)
		assert.Equal([]string{"a", "b"}, ret)

		ret, err = d.Dedupe(ctx, []string{"a", "b", "c"})
		assert.NoError(err)
		assert.Equal([]string{"c"}, ret)
	})

	t.Run("bf", func(t *testing.T) {
		var ctx = context.Background()
		var assert = require.New(t)
		var d = DedupeBF(2000000, 5)

		ret, err := d.Dedupe(ctx, []string{"a", "b"})
		assert.NoError(err)
		assert.Equal([]string{"a", "b"}, ret)

		ret, err = d.Dedupe(ctx, []string{"a", "b", "c"})
		assert.NoError(err)
		assert.Equal([]string{"c"}, ret)
	})
}

func BenchmarkDedupe(b *testing.B) {
	b.Run("map", func(b *testing.B) {
		var ctx = context.Background()
		var urls = [...]string{"a", "b"}
		var d = DedupeMap()

		for i := 0; i < b.N; i++ {
			d.Dedupe(ctx, urls[:])
		}
	})

	b.Run("bf", func(b *testing.B) {
		var ctx = context.Background()
		var urls = [...]string{"a", "b"}
		var d = DedupeBF(200000, 5)

		for i := 0; i < b.N; i++ {
			d.Dedupe(ctx, urls[:])
		}
	})
}

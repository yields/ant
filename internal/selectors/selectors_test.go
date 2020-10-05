package selectors

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSelectors(t *testing.T) {
	t.Run("compile", func(t *testing.T) {
		var assert = require.New(t)
		var cache = NewCache()

		s, err := cache.Compile(`title`)

		assert.NoError(err)
		assert.NotNil(s)
	})

	t.Run("cached", func(t *testing.T) {
		var assert = require.New(t)
		var cache = NewCache()

		_, err := cache.Compile(`title`)
		assert.NoError(err)

		_, err = cache.Compile(`title`)
		assert.NoError(err)
	})

	t.Run("compile error", func(t *testing.T) {
		var assert = require.New(t)
		var cache = NewCache()

		_, err := cache.Compile(`[`)
		assert.Error(err)
	})

	t.Run("global cache", func(t *testing.T) {
		var assert = require.New(t)

		s, err := Compile("title")

		assert.NoError(err)
		assert.NotNil(s)
	})
}

func BenchmarkSelectors(b *testing.B) {
	var cache = NewCache()

	b.Run("compile", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				v, err := cache.Compile("title")
				if err != nil {
					b.Fatalf("compile: %s", err)
				}
				if v == nil {
					b.Fatal("nil selector")
				}
			}
		})
	})
}

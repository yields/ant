package antcache

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMemstore(t *testing.T) {
	t.Run("store, load", func(t *testing.T) {
		var ctx = context.Background()
		var assert = require.New(t)
		var mem = &memstore{}

		err := mem.Store(ctx, uint64(1), []byte("v"))
		assert.NoError(err)

		v, err := mem.Load(ctx, uint64(1))
		assert.NoError(err)
		assert.Equal([]byte("v"), v)
	})

	t.Run("load not found", func(t *testing.T) {
		var ctx = context.Background()
		var assert = require.New(t)
		var mem = &memstore{}

		v, err := mem.Load(ctx, uint64(1))
		assert.NoError(err)
		assert.Nil(v)
	})
}

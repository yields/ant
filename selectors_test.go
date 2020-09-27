package ant

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSelectors(t *testing.T) {
	t.Run("compile", func(t *testing.T) {
		var assert = require.New(t)

		s := selectors.compile(`title`)

		assert.NotNil(s)
	})
}

func BenchmarkSelectors(b *testing.B) {
	b.Run("compile", func(b *testing.B) {
		var sel = `title`

		for i := 0; i < b.N; i++ {
			selectors.compile(sel)
		}
	})
}

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

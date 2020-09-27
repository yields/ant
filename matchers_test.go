package ant

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMatchers(t *testing.T) {
	t.Run("regexp", func(t *testing.T) {
		var assert = require.New(t)
		var regexp = MatchRegexp(`[a-z]+\.example\.com`)

		u, _ := url.Parse("https://foo.example.com")
		assert.True(regexp.Match(u), u.String())

		u, _ = url.Parse("https://example.com")
		assert.False(regexp.Match(u), u.String())
	})

	t.Run("hostname", func(t *testing.T) {
		var assert = require.New(t)
		var host = MatchHostname(`example.com`)

		u, _ := url.Parse("https://foo.example.com")
		assert.False(host.Match(u), u.String())

		u, _ = url.Parse("https://example.com")
		assert.True(host.Match(u), u.String())
	})
}

package ant

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMatchers(t *testing.T) {
	t.Run("hostname", func(t *testing.T) {
		var cases = []struct {
			rawurl  string
			pattern string
			match   bool
		}{
			{"https://foo.example.com", `example.com`, false},
			{"https://example.com", `example.com`, true},
		}

		for _, c := range cases {
			t.Run(c.rawurl, func(t *testing.T) {
				var assert = require.New(t)
				var match = MatchHostname(c.pattern)

				u, err := url.Parse(c.rawurl)
				assert.NoError(err)

				assert.Equal(c.match, match.Match(u))
			})
		}
	})
}

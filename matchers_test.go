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

	t.Run("pattern", func(t *testing.T) {
		var cases = []struct {
			rawurl  string
			pattern string
			match   bool
		}{
			{"http://example.com", `example.com`, true},
			{"https://example.com", `example.com`, true},
			{"https://foo.example.com", `*example.com`, true},
			{"https://example.com/foo/baz", `example.com/foo/*`, true},

			{"https://example.com", `example.com/foo/*`, false},
		}

		for _, c := range cases {
			t.Run(c.rawurl, func(t *testing.T) {
				var assert = require.New(t)
				var match = MatchPattern(c.pattern)

				u, err := url.Parse(c.rawurl)
				assert.NoError(err)

				assert.Equal(c.match, match.Match(u))
			})
		}
	})

	t.Run("regexp", func(t *testing.T) {
		var cases = []struct {
			rawurl  string
			pattern string
			match   bool
		}{
			{"http://example.com", `example\.com`, true},
			{"https://example.com", `example\.com`, true},
			{"https://example.com/foo/baz", `example\.com`, true},
			{"https://example.com/foo?query", `^example\.com\/foo$`, true},
		}

		for _, c := range cases {
			t.Run(c.rawurl, func(t *testing.T) {
				var assert = require.New(t)
				var match = MatchRegexp(c.pattern)

				u, err := url.Parse(c.rawurl)
				assert.NoError(err)

				assert.Equal(c.match, match.Match(u), u)
			})
		}
	})

	t.Run("regexp error", func(t *testing.T) {
		var assert = require.New(t)

		defer func() {
			err, ok := recover().(string)
			assert.True(ok, "expected a panic")
			assert.Contains(err, `ant: regexp "[" - error parsing`)
		}()

		MatchRegexp(`[`)
	})
}

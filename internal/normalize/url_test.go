package normalize

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestURL(t *testing.T) {
	var cases = []struct {
		title  string
		input  string
		output string
	}{
		{
			"Uppercase percent-encoded triplets",
			"http://example.com/foo%2a",
			"http://example.com/foo%2A",
		},
		{
			"Lowercase the scheme and hostname",
			"HTTP://User@Example.COM/Foo",
			"http://User@example.com/Foo",
		},
		{
			"Decode percent-encoded triplets",
			"http://example.com/%7Efoo",
			"http://example.com/~foo",
		},
		{
			"Removes dot segments",
			"http://example.com/foo/./bar/baz/../qux",
			"http://example.com/foo/bar/qux",
		},
		{
			"Converts an empty path to `/`",
			"http://example.com",
			"http://example.com/",
		},
		{
			"Removing the default http port",
			"http://example.com:80/",
			"http://example.com/",
		},
		{
			"Removing the default https port",
			"https://example.com:443/",
			"https://example.com/",
		},
		{
			"Keeps custom ports.",
			"http://example.com:8080/",
			"http://example.com:8080/",
		},
		{
			"Removes `?` when query is empty",
			"http://example.com/?",
			"http://example.com/",
		},
		{
			"Sorts query parameters",
			"http://example.com/?a=1&c=3&b=2",
			"http://example.com/?a=1&b=2&c=3",
		},
		{
			"Remove the fragment",
			"http://example.com/#foo",
			"http://example.com/",
		},
	}

	for _, c := range cases {
		t.Run(c.title, func(t *testing.T) {
			var assert = require.New(t)

			v, err := RawURL(c.input)

			assert.NoError(err)
			assert.Equal(c.output, v)
		})
	}
}

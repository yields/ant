package scan

import (
	"strings"
	"testing"

	"github.com/andybalholm/cascadia"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/html"
)

func TestScanner(t *testing.T) {
	t.Run("type cache", func(t *testing.T) {
		var assert = require.New(t)
		var scanner = NewScanner()
		var src = parse(t, "<i>10</i>")
		var dst int

		err := scanner.Scan(&dst, src, Options{})
		assert.NoError(err)

		assert.Equal(1, len(scanner.types))
	})

	t.Run("scan int", func(t *testing.T) {
		var assert = require.New(t)
		var scanner = NewScanner()
		var src = parse(t, "<i>10</i>")
		var dst int

		err := scanner.Scan(&dst, src, Options{})
		assert.NoError(err)
		assert.Equal(10, dst)
	})

	t.Run("scan uint", func(t *testing.T) {
		var assert = require.New(t)
		var scanner = NewScanner()
		var src = parse(t, "<i>10</i>")
		var dst uint

		err := scanner.Scan(&dst, src, Options{})
		assert.NoError(err)
		assert.Equal(uint(10), dst)
	})

	t.Run("scan float32", func(t *testing.T) {
		var assert = require.New(t)
		var scanner = NewScanner()
		var src = parse(t, "<i>10.5</i>")
		var dst float32

		err := scanner.Scan(&dst, src, Options{})
		assert.NoError(err)
		assert.Equal(float32(10.5), dst)
	})

	t.Run("scan float64", func(t *testing.T) {
		var assert = require.New(t)
		var scanner = NewScanner()
		var src = parse(t, "<i>10.5</i>")
		var dst float64

		err := scanner.Scan(&dst, src, Options{})
		assert.NoError(err)
		assert.Equal(float64(10.5), dst)
	})

	t.Run("scan string", func(t *testing.T) {
		var assert = require.New(t)
		var scanner = NewScanner()
		var src = parse(t, "<i>10.5</i>")
		var dst string

		err := scanner.Scan(&dst, src, Options{})
		assert.NoError(err)
		assert.Equal("10.5", dst)
	})

	t.Run("scan bytes", func(t *testing.T) {
		var assert = require.New(t)
		var scanner = NewScanner()
		var src = parse(t, "<i>bytes</i>")
		var dst []byte

		err := scanner.Scan(&dst, src, Options{})
		assert.NoError(err)
		assert.Equal([]byte("bytes"), dst)
	})

	t.Run("scan slice", func(t *testing.T) {
		var assert = require.New(t)
		var scanner = NewScanner()
		var src = parse(t, "<s>a</s><s>b</s>")
		var dst []string

		err := scanner.Scan(&dst, src, Options{
			Selector: cascadia.MustCompile(`s`),
		})

		assert.NoError(err)
		assert.Equal([]string{"a", "b"}, dst)
	})

	t.Run("scan slice attr", func(t *testing.T) {
		var assert = require.New(t)
		var scanner = NewScanner()
		var src = parse(t, "<a href=a></a> <a href=b></a>")
		var dst []string

		err := scanner.Scan(&dst, src, Options{
			Selector: cascadia.MustCompile(`a`),
			Attr:     "href",
		})

		assert.NoError(err)
		assert.Equal([]string{"a", "b"}, dst)
	})

	t.Run("scan struct", func(t *testing.T) {
		var assert = require.New(t)
		var scanner = NewScanner()
		var src = parse(t, `
			<div>
				<int>5</int>
				<uint>6</uint>
				<float>1.5</float>
				<string>foo</string>
			</div>
		`)

		var dst struct {
			Int    int     `css:"int"`
			Uint   uint    `css:"uint"`
			Float  float32 `css:"float"`
			String string  `css:"string"`
		}

		err := scanner.Scan(&dst, src, Options{})
		assert.NoError(err)

		assert.Equal(5, dst.Int)
		assert.Equal(uint(6), dst.Uint)
		assert.Equal(float32(1.5), dst.Float)
		assert.Equal("foo", dst.String)
	})

	t.Run("scan nested struct", func(t *testing.T) {
		var assert = require.New(t)
		var scanner = NewScanner()
		var src = parse(t, `
			<div>
				<struct>
					<int>5</int>
					<uint>6</uint>
					<float>1.5</float>
					<string>foo</string>
				</struct>
			</div>
		`)

		var dst struct {
			Struct struct {
				Int    int     `css:"int"`
				Uint   uint    `css:"uint"`
				Float  float32 `css:"float"`
				String string  `css:"string"`
			} `css:"struct"`
		}

		err := scanner.Scan(&dst, src, Options{})
		assert.NoError(err)

		assert.Equal(5, dst.Struct.Int)
		assert.Equal(uint(6), dst.Struct.Uint)
		assert.Equal(float32(1.5), dst.Struct.Float)
		assert.Equal("foo", dst.Struct.String)
	})
}

func parse(t testing.TB, s string) *html.Node {
	t.Helper()

	root, err := html.Parse(strings.NewReader(s))
	if err != nil {
		t.Fatalf("parse: %s", err)
	}

	return root
}

package ant

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"golang.org/x/net/html"
)

func TestList(t *testing.T) {
	t.Run("query", func(t *testing.T) {
		var assert = require.New(t)
		var root = parse(t, `<title>title</title>`)
		var list = List{root}

		list = list.Query(`title`)

		assert.Equal(1, len(list))
		assert.Equal("title", list[0].DataAtom.String())
	})

	t.Run("text", func(t *testing.T) {
		var assert = require.New(t)
		var root = parse(t, `<title>title</title>`)
		var list = List{root}.Query("title")

		assert.Equal("title", list.Text())
	})

	t.Run("text empty", func(t *testing.T) {
		var assert = require.New(t)

		assert.Equal("", List{}.Text())
	})

	t.Run("attr", func(t *testing.T) {
		var assert = require.New(t)
		var root = parse(t, `<title key=val></title>`)
		var list = List{root}.Query("title")

		v, ok := list.Attr("key")

		assert.True(ok)
		assert.Equal("val", v)
	})

	t.Run("attr empty", func(t *testing.T) {
		var assert = require.New(t)

		v, ok := List{}.Attr("foo")

		assert.False(ok)
		assert.Equal("", v)
	})

	t.Run("scan struct", func(t *testing.T) {
		var assert = require.New(t)
		var root = parse(t, `<title key=val>title</title>`)
		var list = List{root}.Query("title")
		var data struct {
			Title string `css:"title"`
			Key   string `css:"title@key"`
		}

		err := list.Scan(&data)
		assert.NoError(err)

		assert.Equal("title", data.Title)
		assert.Equal("val", data.Key)
	})

	t.Run("scan empty", func(t *testing.T) {
		var assert = require.New(t)
		var data struct {
			Title string `css:"title"`
			Key   string `css:"title@key"`
		}

		err := List{}.Scan(&data)
		assert.NoError(err)
	})
}

func BenchmarkList(b *testing.B) {
	b.Run("query", func(b *testing.B) {
		var root = parse(b, `<title>title</title>`)
		var list = List{root}

		for i := 0; i < b.N; i++ {
			list = list.Query(`title`)
		}
	})

	b.Run("text", func(b *testing.B) {
		var root = parse(b, `<title>title</title>`)
		var list = List{root}.Query("title")

		for i := 0; i < b.N; i++ {
			list.Text()
		}
	})

	b.Run("attr", func(b *testing.B) {
		var root = parse(b, `<title key=val></title>`)
		var list = List{root}.Query("title")

		for i := 0; i < b.N; i++ {
			list.Attr(`title`)
		}
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

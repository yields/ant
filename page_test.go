package ant

import (
	"io"
	"io/ioutil"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPage(t *testing.T) {
	t.Run("urls", func(t *testing.T) {
		var page = makePage(t, `
			<a href="/foo">foo</a>
			<a href="https://foo.com">abs</a>
		`)
		var assert = require.New(t)

		all := page.URLs()

		assert.Equal(2, len(all))
		assert.Equal("https://example.com/foo", all[0].String())
		assert.Equal("https://foo.com", all[1].String())
	})

	t.Run("text", func(t *testing.T) {
		var page = makePage(t, `<title>foo</title>`)
		var assert = require.New(t)

		text := page.Text("title")

		assert.Equal("foo", text)
	})

	t.Run("scan", func(t *testing.T) {
		var page = makePage(t, `<name>ant</name><stars>9`)
		var assert = require.New(t)
		var repo struct {
			Name  string `css:"name"`
			Stars int    `css:"stars"`
		}

		err := page.Scan(&repo)

		assert.NoError(err)
		assert.Equal("ant", repo.Name)
		assert.Equal(9, repo.Stars)
	})

	t.Run("scan invalid HTML", func(t *testing.T) {
		var u, _ = url.Parse("https://example.com")
		var page = &Page{URL: u, body: readerError{}}
		var assert = require.New(t)
		var repo struct {
			Name  string `css:"name"`
			Stars int    `css:"stars"`
		}

		err := page.Scan(&repo)
		assert.Error(err)
		assert.EqualError(err, `ant: parse html "https://example.com" - short buffer`)
	})
}

func BenchmarkPage(b *testing.B) {
	var buf = `<a href="/foo">foo</a>`

	b.Run("urls", func(b *testing.B) {
		var p = makePage(b, buf)

		for i := 0; i < b.N; i++ {
			p.URLs()
		}
	})

	b.Run("text", func(b *testing.B) {
		var p = makePage(b, `<title>foo</title>`)

		for i := 0; i < b.N; i++ {
			p.Text("title")
		}
	})

	b.Run("query", func(b *testing.B) {
		var p = makePage(b, `<title>foo</title>`)

		for i := 0; i < b.N; i++ {
			p.Query("title")
		}
	})
}

func makePage(t testing.TB, buf string) *Page {
	t.Helper()
	u, _ := url.Parse("https://example.com")
	r := strings.NewReader(buf)
	return &Page{
		URL:  u,
		body: ioutil.NopCloser(r),
	}
}

type readerError struct{}

func (readerError) Read(p []byte) (n int, err error) {
	err = io.ErrShortBuffer
	return
}

func (readerError) Close() error {
	return nil
}

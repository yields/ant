package antcache

import (
	"context"
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestDiskstore(t *testing.T) {
	t.Run("open empty path", func(t *testing.T) {
		var assert = require.New(t)

		_, err := Open("")

		assert.EqualError(err, `antcache: disk expects an absolute path, got ""`)
	})

	t.Run("open not absolute path", func(t *testing.T) {
		var assert = require.New(t)

		_, err := Open("tmp")

		assert.EqualError(err, `antcache: disk expects an absolute path, got "tmp"`)
	})

	t.Run("open missing", func(t *testing.T) {
		var assert = require.New(t)

		_, err := Open("/tmp/foo")

		assert.EqualError(err, `antcache: disk open /tmp/foo: no such file or directory`)
	})

	t.Run("open not dir", func(t *testing.T) {
		var assert = require.New(t)

		_, err := Open(tempfile(t))

		assert.EqualError(err, `antcache: disk expected a directory`)
	})

	t.Run("open", func(t *testing.T) {
		var assert = require.New(t)

		d, err := Open(tempdir(t))

		assert.NoError(err)
		assert.NoError(d.Close())
	})

	t.Run("wait", func(t *testing.T) {
		var ctx = context.Background()
		var assert = require.New(t)

		d, err := Open(tempdir(t))
		assert.NoError(err)

		err = d.Wait(ctx)
		assert.NoError(err)

		err = d.Close()
		assert.NoError(err)
	})

	t.Run("wait canceled", func(t *testing.T) {
		var ctx = context.Background()
		var assert = require.New(t)

		d, err := Open(tempdir(t))
		assert.NoError(err)

		ctx, cancel := context.WithCancel(ctx)
		cancel()

		err = d.Wait(ctx)
		assert.Equal(context.Canceled, err)

		err = d.Close()
		assert.NoError(err)
	})

	t.Run("store and load", func(t *testing.T) {
		var ctx = context.Background()
		var assert = require.New(t)

		d, err := Open(tempdir(t))
		assert.NoError(err)

		err = d.Store(ctx, 0, []byte("yo"))
		assert.NoError(err)

		v, err := d.Load(ctx, 0)
		assert.NoError(err)
		assert.Equal([]byte("yo"), v)
	})

	t.Run("store and load compressed", func(t *testing.T) {
		var ctx = context.Background()
		var assert = require.New(t)

		d, err := Open(tempdir(t), Compress())
		assert.NoError(err)

		err = d.Store(ctx, 0, []byte("yo"))
		assert.NoError(err)

		v, err := d.Load(ctx, 0)
		assert.NoError(err)
		assert.Equal([]byte("yo"), v)
	})

	t.Run("store un-compressed, load compressed", func(t *testing.T) {
		var ctx = context.Background()
		var assert = require.New(t)
		var root = tempdir(t)

		d, err := Open(root)
		assert.NoError(err)

		err = d.Store(ctx, 0, []byte("yo"))
		assert.NoError(err)
		assert.NoError(d.Close())

		d, err = Open(root, Compress())
		assert.NoError(err)
		assert.NoError(d.Wait(ctx))

		v, err := d.Load(ctx, 0)
		assert.Error(err)
		assert.Contains(err.Error(), "antcache: compress is on but snappy can't decode")
		assert.True(nil == v)
		assert.NoError(d.Close())
	})

	t.Run("when maxage is set, expired files are removed", func(t *testing.T) {
		var ctx = context.Background()
		var assert = require.New(t)

		d, err := Open(tempdir(t), Maxage(1*time.Second))
		assert.NoError(err)

		d.Store(ctx, 0, []byte("one"))
		d.Store(ctx, 1, []byte("two"))
		assert.Equal(2, len(d.files()))

		d.now = func() time.Time {
			return time.Now().Add(2 * time.Second)
		}

		n, err := d.sweep()
		assert.NoError(err)
		assert.Equal(2, n)

		assert.Equal(0, len(d.files()))
	})

	t.Run("when maxsize is set, files are removed", func(t *testing.T) {
		var ctx = context.Background()
		var assert = require.New(t)

		d, err := Open(tempdir(t), Maxsize(4))
		assert.NoError(err)

		d.Store(ctx, 0, []byte("one"))
		d.Store(ctx, 1, []byte("two"))
		assert.Equal(2, len(d.files()))

		n, err := d.sweep()
		assert.NoError(err)
		assert.Equal(1, n)

		t.Logf("files: %+v\n", d.files())
		assert.Equal(1, len(d.files()))
	})

	t.Run("when re-opened, it fetches all files from disk", func(t *testing.T) {
		var ctx = context.Background()
		var assert = require.New(t)
		var root = tempdir(t)

		d, err := Open(root)
		assert.NoError(err)

		d.Store(ctx, 0, []byte("one"))
		d.Store(ctx, 1, []byte("two"))
		assert.Equal(2, len(d.files()))
		assert.NoError(d.Close())

		d, err = Open(root)
		assert.NoError(err)

		err = d.Wait(ctx)
		assert.NoError(err)

		assert.Equal(2, len(d.files()))
	})
}

func tempdir(t testing.TB) string {
	t.Helper()

	name, err := ioutil.TempDir("", "antcache_disktest.*")
	if err != nil {
		t.Fatalf("tempdir: %s", err)
	}

	t.Cleanup(func() {
		os.RemoveAll(name)
	})

	return name
}

func tempfile(t testing.TB) string {
	t.Helper()

	f, err := ioutil.TempFile("", "antcache_disktest.*")
	if err != nil {
		t.Fatalf("tmpfile: %s", err)
	}
	f.Close()

	t.Cleanup(func() {
		os.Remove(f.Name())
	})

	return f.Name()
}

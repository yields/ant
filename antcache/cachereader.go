package antcache

import (
	"bytes"
	"context"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httputil"
	"sync"
)

// Cachereader is a special reader that caches
// a response when it is closed.
type cachereader struct {
	resp  *http.Response
	key   uint64
	rc    io.ReadCloser
	buf   *bytes.Buffer
	once  *sync.Once
	ctx   context.Context
	store func(ctx context.Context, key uint64, v []byte) error
	log   func(msg string, args ...interface{})
}

// Read implementation.
func (cr *cachereader) Read(p []byte) (n int, err error) {
	n, err = cr.rc.Read(p)
	cr.buf.Write(p)
	return n, err
}

// Close implementation.
func (cr *cachereader) Close() error {
	err := cr.rc.Close()

	cr.once.Do(func() {
		resp := *cr.resp
		resp.Body = ioutil.NopCloser(cr.buf)

		buf, err := httputil.DumpResponse(&resp, true)
		if err != nil {
			cr.log("antcache: dump response - %s", err)
			return
		}

		if err := cr.store(cr.ctx, cr.key, buf); err != nil {
			cr.log("antcache: store response - %s", err)
		}
	})

	return err
}

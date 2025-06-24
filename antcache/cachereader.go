package antcache

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"sync"

	"github.com/hashicorp/go-multierror"
)

// Cachereader is a special reader that caches
// a response when it is closed.
type cachereader struct {
	resp  *http.Response
	key   uint64
	rc    io.ReadCloser
	buf   bytes.Buffer
	once  sync.Once
	ctx   context.Context
	store func(ctx context.Context, key uint64, v []byte) error
}

// Read implementation.
func (cr *cachereader) Read(p []byte) (n int, err error) {
	if n, err = cr.rc.Read(p); n > 0 {
		cr.buf.Write(p[:n])
	}
	return n, err
}

// Close implementation.
func (cr *cachereader) Close() error {
	cerr := cr.rc.Close()

	cr.once.Do(func() {
		resp := *cr.resp

		r := bytes.NewReader(cr.buf.Bytes())
		resp.Body = io.NopCloser(r)

		buf, err := httputil.DumpResponse(&resp, true)
		if err != nil {
			cerr = multierror.Append(cerr, fmt.Errorf(
				`antcache: dump response %q - %w`,
				resp.Request.URL,
				err,
			))
			return
		}

		if err := cr.store(cr.ctx, cr.key, buf); err != nil {
			cerr = multierror.Append(cerr, fmt.Errorf(
				`antcache: store %d - %w`,
				cr.key,
				err,
			))
		}
	})

	return cerr
}

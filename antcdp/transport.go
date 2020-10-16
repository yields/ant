package antcdp

import (
	"net/http"

	"github.com/hashicorp/go-multierror"
)

// Transport implements a CDP request transport.
//
// The transport manages a pool of targets, it removes idle
// targets and creates targets as configured. When a request
// arrives the transport translates it to CDP commands that
// acquire and reset an idle target, set the cookies and headers
// and send the request.
//
// A transport is safe to use from multiple goroutines.
type transport struct {
	pool *targets
}

// Roundtrip performs a roundtrip.
func (t *transport) roundtrip(req *http.Request) (*http.Response, error) {
	var ctx = req.Context()

	target, err := t.pool.acquire(ctx)
	if err != nil {
		return nil, err
	}
	defer t.pool.release(target)

	tx := &tx{
		request: req,
		target:  target,
	}

	if err := tx.init(ctx); err != nil {
		return nil, err
	}

	defer func() {
		err = multierror.Append(err, tx.close())
	}()

	resp, err := tx.do(ctx)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

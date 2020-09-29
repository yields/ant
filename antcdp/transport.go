package antcdp

import "net/http"

// Transport implements a transport.
type Transport struct{}

// RoundTrip implementation.
func (t *Transport) RoundTrip(req *http.Request) (*http.Response, error) {
	return nil, nil
}

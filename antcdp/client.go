// Package antcdp is an experimental package that implements an `ant.Client`
// that performs HTTP requests using chrome and returns a rendered response.
//
// Usage:
//
//   eng, err := ant.NewEngine(ant.EngineConfig{
//     Fetcher: &ant.Fetcher{
//       Client: &antcdp.Client{},
//     }
//   })
//
package antcdp

import (
	"net/http"
	"sync"

	"github.com/mafredri/cdp/devtool"
)

const (
	// Addr is the default address to connect to.
	//
	// It is used if `Client.Addr` is empty.
	Addr = "http://127.0.0.1:9222"
)

// Client implements a chrome debugger protocol client.
//
// The client is similar to the default net/http.Client
// it receives http.Request translates it to CDP commands
// and returns an http.Response.
//
// Its zero-value is ready for use and connects to a CDP
// server locally at `127.0.0.1:9222`.
type Client struct {
	// Addr is the address to connect to.
	//
	// If empty, it defaults to `antcdp.Addr`.
	Addr string

	// Transport is initialized on the 1st request.
	transport *transport
	once      sync.Once
}

// Do implementation.
func (c *Client) Do(req *http.Request) (*http.Response, error) {
	c.once.Do(func() {
		c.transport = &transport{
			pool: newTargets(devtool.New(c.addr())),
		}
	})

	return c.transport.roundtrip(req)
}

// Addr returns the address.
func (c *Client) addr() string {
	if c.Addr != "" {
		return c.Addr
	}
	return Addr
}

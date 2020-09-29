// Package antcdp implements an ant client that uses CDP.
//
// Usage:
//
//   eng, err := ant.NewEngine(ant.EngineConfig{
//     Fetcher: &ant.Fetcher{
//       Client: antcdp.Client{},
//     }
//   })
//
package antcdp

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/mafredri/cdp"
	"github.com/mafredri/cdp/devtool"
	"github.com/mafredri/cdp/protocol/dom"
	"github.com/mafredri/cdp/protocol/page"
	"github.com/mafredri/cdp/rpcc"
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
}

// Do implementation.
func (c *Client) Do(req *http.Request) (*http.Response, error) {
	var ctx = req.Context()

	t, err := c.target(ctx)
	if err != nil {
		return nil, err
	}

	sock, err := rpcc.DialContext(ctx, t.WebSocketDebuggerURL)
	if err != nil {
		return nil, fmt.Errorf("antcdp: dial %q - %w",
			t.WebSocketDebuggerURL,
			err,
		)
	}
	defer sock.Close()

	client := cdp.NewClient(sock)
	ready, err := client.Page.DOMContentEventFired(ctx)
	if err != nil {
		return nil, fmt.Errorf("antcdp: ready - %w", err)
	}
	defer ready.Close()

	err = client.Page.Enable(ctx)
	if err != nil {
		return nil, fmt.Errorf("antcdp: enable - %w", err)
	}

	args := page.NewNavigateArgs(req.URL.String())

	_, err = client.Page.Navigate(ctx, args)
	if err != nil {
		return nil, fmt.Errorf("antcdp: navigate - %w", err)
	}

	if _, err := ready.Recv(); err != nil {
		return nil, fmt.Errorf("antcdp: ready wait - %w", err)
	}

	doc, err := client.DOM.GetDocument(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("antcdp: get document - %w", err)
	}

	resp, err := client.DOM.GetOuterHTML(ctx, &dom.GetOuterHTMLArgs{
		NodeID: &doc.Root.NodeID,
	})
	if err != nil {
		return nil, fmt.Errorf("antcdp: outer html - %w", err)
	}

	body := strings.NewReader(resp.OuterHTML)

	return &http.Response{
		Request:       req,
		Body:          ioutil.NopCloser(body),
		ContentLength: int64(len(resp.OuterHTML)),
		Status:        "200 OK",
		Proto:         "http/1.1",
		ProtoMajor:    1,
		ProtoMinor:    1,
		StatusCode:    200,
		Uncompressed:  true,
	}, nil
}

// Target returns a target.
func (c *Client) target(ctx context.Context) (*devtool.Target, error) {
	var d = devtool.New(c.addr())

	t, err := d.Get(ctx, devtool.Page)

	if err != nil {
		t, err = d.Create(ctx)
	}

	if err != nil {
		err = fmt.Errorf("antcdp: target - %w", err)
	}

	return t, err
}

// Addr returns the address.
func (c *Client) addr() string {
	if c.Addr != "" {
		return c.Addr
	}
	return Addr
}

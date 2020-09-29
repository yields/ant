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
	"strconv"
	"strings"

	"github.com/mafredri/cdp"
	"github.com/mafredri/cdp/devtool"
	"github.com/mafredri/cdp/protocol/dom"
	"github.com/mafredri/cdp/protocol/network"
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

	resp, err := client.Network.ResponseReceived(ctx)
	if err != nil {
		return nil, fmt.Errorf("antcdp: response received - %w", err)
	}
	defer resp.Close()

	if err := client.Network.Enable(ctx, nil); err != nil {
		return nil, fmt.Errorf("antcdp: network enable - %w", err)
	}

	args := page.NewNavigateArgs(req.URL.String())
	reply, err := client.Page.Navigate(ctx, args)
	if err != nil {
		return nil, fmt.Errorf("antcdp: navigate - %w", err)
	}

	if errmsg := reply.ErrorText; errmsg != nil {
		return nil, fmt.Errorf("antcdp: navigate - %s", *errmsg)
	}

	var (
		status = 200
		hdr    = make(http.Header)
		proto  string
	)

	for {
		event, err := resp.Recv()
		if err != nil {
			return nil, fmt.Errorf("antcdp: response recv - %w", err)
		}

		if event.Type != network.ResourceTypeDocument {
			continue
		}

		if id := event.FrameID; id != nil && *id != reply.FrameID {
			continue
		}

		status = event.Response.Status

		if p := event.Response.Protocol; p != nil {
			proto = *p
		}

		headers, err := event.Response.Headers.Map()
		if err != nil {
			return nil, fmt.Errorf("antcdp: parse headers - %w", err)
		}

		for k, v := range headers {
			hdr.Set(k, v)
		}

		break
	}

	if _, err := ready.Recv(); err != nil {
		return nil, fmt.Errorf("antcdp: ready wait - %w", err)
	}

	doc, err := client.DOM.GetDocument(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("antcdp: get document - %w", err)
	}

	content, err := client.DOM.GetOuterHTML(ctx, &dom.GetOuterHTMLArgs{
		NodeID: &doc.Root.NodeID,
	})
	if err != nil {
		return nil, fmt.Errorf("antcdp: outer html - %w", err)
	}

	var (
		body = strings.NewReader(content.OuterHTML)
	)

	major, minor, ok := parseProto(proto)
	if !ok {
		return nil, fmt.Errorf("antcdp: invalid protocol %q", proto)
	}

	hdr.Set("Content-Length", strconv.Itoa(len(content.OuterHTML)))

	return &http.Response{
		Request:       req,
		Body:          ioutil.NopCloser(body),
		ContentLength: int64(len(content.OuterHTML)),
		Status:        fmt.Sprintf("%d %s", status, http.StatusText(status)),
		Proto:         proto,
		ProtoMajor:    major,
		ProtoMinor:    minor,
		StatusCode:    status,
		Uncompressed:  true,
		Header:        hdr,
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

// ParseProto parses a protocol s.
func parseProto(s string) (major, minor int, ok bool) {
	if j := strings.IndexByte(s, '/'); j != -1 {
		if p := strings.SplitN(s[j+1:], ".", 2); len(p) == 2 {
			major, _ = strconv.Atoi(p[0])
			minor, _ = strconv.Atoi(p[1])
			ok = true
		}
	}
	return
}

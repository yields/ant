package antcdp

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/hashicorp/go-multierror"
	"github.com/mafredri/cdp"
	"github.com/mafredri/cdp/devtool"
	"github.com/mafredri/cdp/protocol/dom"
	"github.com/mafredri/cdp/protocol/network"
	"github.com/mafredri/cdp/protocol/page"
	"github.com/mafredri/cdp/protocol/storage"
	"github.com/mafredri/cdp/rpcc"
)

// Tx represents a single transaction.
type tx struct {
	target  *devtool.Target
	request *http.Request
	resp    *http.Response
	conn    *rpcc.Conn
	client  *cdp.Client
	events  struct {
		req   network.RequestWillBeSentClient
		res   network.ResponseReceivedClient
		ready page.DOMContentEventFiredClient
	}
}

// Init initializes the transaction.
func (tx *tx) init(ctx context.Context) error {
	var url = tx.target.WebSocketDebuggerURL

	conn, err := rpcc.DialContext(ctx, url)
	if err != nil {
		return fmt.Errorf("antcdp: dial %q - %w", url, err)
	}

	tx.conn = conn
	tx.resp = &http.Response{Request: tx.request}
	tx.client = cdp.NewClient(conn)

	if err := tx.client.Page.Enable(ctx); err != nil {
		return err
	}

	if err := tx.client.Network.Enable(ctx, nil); err != nil {
		return err
	}

	if err := tx.setHeaders(ctx); err != nil {
		return err
	}

	if err := tx.setCookies(ctx); err != nil {
		return err
	}

	reqc, err := tx.client.Network.RequestWillBeSent(ctx)
	if err != nil {
		return err
	}

	resc, err := tx.client.Network.ResponseReceived(ctx)
	if err != nil {
		return err
	}

	ready, err := tx.client.Page.DOMContentEventFired(ctx)
	if err != nil {
		return err
	}

	tx.events.req = reqc
	tx.events.res = resc
	tx.events.ready = ready
	return nil
}

// Do sends the navigates to the page.
func (tx *tx) do(ctx context.Context) (*http.Response, error) {
	var args = page.NewNavigateArgs(tx.request.URL.String())

	reply, err := tx.client.Page.Navigate(ctx, args)
	if err != nil {
		return nil, fmt.Errorf("antcdp: navigate - %w", err)
	}

	if errmsg := reply.ErrorText; errmsg != nil {
		return nil, fmt.Errorf("antcdp: navigate - %s", *errmsg)
	}

	if err := tx.wait(ctx, *reply); err != nil {
		return nil, err
	}

	if _, err := tx.events.ready.Recv(); err != nil {
		return nil, fmt.Errorf("antcdp: dom ready - %w", err)
	}

	if err := tx.readbody(ctx); err != nil {
		return nil, err
	}

	if err := tx.readcookies(ctx); err != nil {
		return nil, err
	}

	return tx.resp, nil
}

// Wait waits for the network requests to load.
func (tx *tx) wait(ctx context.Context, args page.NavigateReply) error {
	var ids = make(map[network.RequestID]struct{})
	var reqc = tx.events.req
	var resc = tx.events.res

	for {
		select {
		case <-reqc.Ready():
			event, err := reqc.Recv()
			if err != nil {
				return fmt.Errorf("antcdp: request recv - %w", err)
			}
			ids[event.RequestID] = struct{}{}

		case <-resc.Ready():
			event, err := resc.Recv()
			if err != nil {
				return fmt.Errorf("antcdp: response recv - %w", err)
			}

			if event.Type != network.ResourceTypeDocument {
				continue
			}
			if id := event.FrameID; id != nil && *id != args.FrameID {
				continue
			}

			if err := tx.merge(event.Response); err != nil {
				return fmt.Errorf("antcdp: merge response - %w", err)
			}

			delete(ids, event.RequestID)
			if len(ids) == 0 {
				return nil
			}

		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

// Readbody attempts to read the page's body.
func (tx *tx) readbody(ctx context.Context) error {
	doc, err := tx.client.DOM.GetDocument(ctx, nil)
	if err != nil {
		return fmt.Errorf("antcdp: get document - %w", err)
	}

	args := dom.GetOuterHTMLArgs{}
	args.NodeID = &doc.Root.NodeID
	reply, err := tx.client.DOM.GetOuterHTML(ctx, &args)
	if err != nil {
		return fmt.Errorf("antcdp: outer html - %w", err)
	}

	if tx.resp.Header == nil {
		tx.resp.Header = make(http.Header)
	}

	var (
		content = reply.OuterHTML
		size    = len(content)
		length  = strconv.Itoa(size)
		body    = io.NopCloser(strings.NewReader(content))
	)

	tx.resp.Body = body
	tx.resp.ContentLength = int64(size)
	tx.resp.Header.Set("Content-Length", length)
	return nil
}

// Readcookies reads the cookies from CDP.
func (tx *tx) readcookies(ctx context.Context) error {
	var storage = tx.client.Storage

	reply, err := storage.GetCookies(ctx, nil)
	if err != nil {
		return fmt.Errorf("antcdp: get cookies - %w", err)
	}

	for _, c := range reply.Cookies {
		cookie := http.Cookie{
			Name:     c.Name,
			Value:    c.Value,
			Domain:   c.Domain,
			Path:     c.Path,
			HttpOnly: c.HTTPOnly,
		}
		tx.resp.Header.Add("Set-Cookie", cookie.String())
	}

	return nil
}

// Merge merges the given response into tx.resp.
func (tx *tx) merge(resp network.Response) error {
	tx.resp.StatusCode = resp.Status
	tx.resp.Status = fmt.Sprintf("%d %s", resp.Status, http.StatusText(resp.Status))

	hdr, err := resp.Headers.Map()
	if err != nil {
		return fmt.Errorf("antcdp: headers map - %w", err)
	}

	for k, v := range hdr {
		if tx.resp.Header == nil {
			tx.resp.Header = make(http.Header)
		}
		tx.resp.Header.Set(k, v)
	}

	if p := resp.Protocol; p != nil {
		major, minor, _ := parseProto(*p)
		tx.resp.Proto = *p
		tx.resp.ProtoMajor = major
		tx.resp.ProtoMinor = minor
	}

	tx.resp.Uncompressed = true
	return nil
}

// SetHeaders sets copies headers from the request to chrome.
func (tx *tx) setHeaders(ctx context.Context) error {
	var args network.SetExtraHTTPHeadersArgs
	var dst = make(map[string]string)
	var src = tx.request.Header

	for k := range src {
		dst[k] = src.Get(k)
	}

	buf, err := json.Marshal(dst)
	if err != nil {
		return fmt.Errorf("antcdp: marshal headers - %w", err)
	}

	args.Headers = network.Headers(buf)
	err = tx.client.Network.SetExtraHTTPHeaders(ctx, &args)
	if err != nil {
		return fmt.Errorf("antcdp: set extra headers - %w", err)
	}

	return nil
}

// SetCookies sets the cookies.
func (tx *tx) setCookies(ctx context.Context) error {
	var cookies = tx.request.Cookies()
	var url = tx.request.URL.String()
	var c = tx.client.Storage
	var args storage.SetCookiesArgs

	args.Cookies = make([]network.CookieParam, len(cookies))

	for j, c := range cookies {
		args.Cookies[j] = network.CookieParam{
			Name:     c.Name,
			Value:    c.Value,
			URL:      &url,
			Domain:   &c.Domain,
			Path:     &c.Path,
			Secure:   &c.Secure,
			HTTPOnly: &c.HttpOnly,
		}
	}

	if err := c.SetCookies(ctx, &args); err != nil {
		return fmt.Errorf("antcdp: set cookies - %w", err)
	}

	return nil
}

// Close closes the transaction.
func (tx *tx) close() (err error) {
	var closers = [...]io.Closer{
		tx.conn,
		tx.events.req,
		tx.events.res,
		tx.events.ready,
	}

	for _, c := range closers {
		if c != nil {
			if e := c.Close(); e != nil {
				err = multierror.Append(err, e)
			}
		}
	}

	return nil
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

package ant

import (
	"github.com/yields/ant/internal/scan"
	"golang.org/x/net/html"
)

var (
	// Scanner represents a node scanner.
	scanner = scan.NewScanner()
)

// List represents a list of nodes.
//
// The list wraps the html node slice with
// helper methods to extract data and manipulate
// the list.
type List []*html.Node

// Query returns a list of nodes matching selector.
//
// If the selector is invalid, the method returns a nil list.
func (l List) Query(selector string) List {
	var ret List

	if sel := selectors.compile(selector); sel != nil {
		for _, n := range l {
			ret = append(ret, sel.MatchAll(n)...)
		}
	}

	return ret
}

// Is returns true if any of the nodes matches selector.
func (l List) Is(selector string) (matched bool) {
	if sel := selectors.compile(selector); sel != nil {
		for _, n := range l {
			if sel.Match(n) {
				matched = true
				break
			}
		}
	}
	return
}

// Text returns inner text of the first node..
func (l List) Text() string {
	for _, n := range l {
		return scan.Text(n)
	}
	return ""
}

// Attr returns the attribute value of key of the first node.
func (l List) Attr(key string) (string, bool) {
	for _, n := range l {
		return scan.Attr(n, key)
	}
	return "", false
}

// Scan scans all items into struct `dst`.
//
// The method scans data from the 1st node.
func (l List) Scan(dst interface{}) error {
	for _, n := range l {
		return scanner.Scan(dst, n, scan.Options{})
	}
	return nil
}

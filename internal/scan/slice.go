package scan

import (
	"reflect"

	"github.com/andybalholm/cascadia"
	"golang.org/x/net/html"
)

// SliceScanner implements a slice scanner.
type sliceScanner struct {
	selector  cascadia.Selector
	scanFunc  ScanFunc
	sliceType reflect.Type
}

// Slice returns a new slice scanner.
func Slice(opts Options, t reflect.Type) (ScanFunc, error) {
	var eltype = t.Elem()

	f, err := ScannerOf(eltype, Options{
		Selector: nil,
		Attr:     opts.Attr,
	})
	if err != nil {
		return nil, err
	}

	return (sliceScanner{
		selector:  opts.Selector,
		scanFunc:  f,
		sliceType: t,
	}).scan, nil
}

// Scan implements a slice scanner.
func (ss sliceScanner) scan(dst reflect.Value, src *html.Node) error {
	var nodes = ss.selector.MatchAll(src)

	if len(nodes) == 0 {
		return nil
	}

	slice := reflect.MakeSlice(
		ss.sliceType,
		len(nodes),
		len(nodes),
	)

	for j, node := range nodes {
		if err := ss.scanFunc(slice.Index(j), node); err != nil {
			return err
		}
	}

	dst.Set(slice)
	return nil
}

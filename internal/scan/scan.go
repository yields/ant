package scan

import (
	"fmt"
	"reflect"
	"strconv"

	"github.com/andybalholm/cascadia"
	"golang.org/x/net/html"
)

// Cached type identities.
var (
	byteslice = reflect.TypeOf([]byte(nil))
)

// Options represents the scan options.
type Options struct {
	Selector cascadia.Selector
	Attr     string
}

// ScanFunc represents a scanner func.
//
// The function receives a destination to scan into and
// the source as the root html node.
//
// Scan functions can be nested, for example a slice scan func
// will typically nest its element types, a struct scan func
// will nest its field types.
type ScanFunc func(dst reflect.Value, n *html.Node) error

// ScannerOf returns a scanner of type t using opts.
//
// If the type is not supported, the method returns
// an error with type information.
//
// Some types which define tags will return an error
// if the tag cannot be compiled.
func ScannerOf(t reflect.Type, opts Options) (ScanFunc, error) {
	switch t.Kind() {
	case reflect.String:
		return String(opts), nil

	case reflect.Int:
		return Int(opts), nil

	case reflect.Uint:
		return Uint(opts), nil

	case reflect.Float32, reflect.Float64:
		return Float(opts), nil

	case reflect.Struct:
		return Struct(opts, t)

	case reflect.Slice:
		if t == byteslice {
			return Bytes(opts), nil
		}
		return Slice(opts, t)
	}

	return nil, fmt.Errorf("scan: cannot scan into type %s", t)
}

// String returns a scanner func for a string.
func String(opts Options) ScanFunc {
	return func(dst reflect.Value, src *html.Node) error {
		if opts.Selector != nil {
			src = opts.Selector.MatchFirst(src)
		}

		if opts.Attr != "" {
			t, _ := Attr(src, opts.Attr)
			dst.SetString(t)
		} else {
			dst.SetString(Text(src))
		}

		return nil
	}
}

// Int returns a scanner func for an int.
func Int(opts Options) ScanFunc {
	return func(dst reflect.Value, src *html.Node) error {
		var str string

		if opts.Selector != nil {
			src = opts.Selector.MatchFirst(src)
		}

		if opts.Attr != "" {
			str, _ = Attr(src, opts.Attr)
		} else {
			str = Text(src)
		}

		x, _ := strconv.ParseInt(str, 10, 64)
		dst.SetInt(x)
		return nil
	}
}

// Uint returns a scanner func for an uint.
func Uint(opts Options) ScanFunc {
	return func(dst reflect.Value, src *html.Node) error {
		var str string

		if opts.Selector != nil {
			src = opts.Selector.MatchFirst(src)
		}

		if opts.Attr != "" {
			str, _ = Attr(src, opts.Attr)
		} else {
			str = Text(src)
		}

		x, _ := strconv.ParseUint(str, 10, 64)
		dst.SetUint(x)
		return nil
	}
}

// Float returns a scanner func for a float.
func Float(opts Options) ScanFunc {
	return func(dst reflect.Value, src *html.Node) error {
		var str string

		if opts.Selector != nil {
			src = opts.Selector.MatchFirst(src)
		}

		if opts.Attr != "" {
			str, _ = Attr(src, opts.Attr)
		} else {
			str = Text(src)
		}

		x, _ := strconv.ParseFloat(str, 64)
		dst.SetFloat(x)
		return nil
	}
}

// Bytes returns a scanner func for a byte slice.
func Bytes(opts Options) ScanFunc {
	return func(dst reflect.Value, src *html.Node) error {
		var str string

		if opts.Selector != nil {
			src = opts.Selector.MatchFirst(src)
		}

		if opts.Attr != "" {
			str, _ = Attr(src, opts.Attr)
		} else {
			str = Text(src)
		}

		dst.SetBytes([]byte(str))
		return nil
	}
}

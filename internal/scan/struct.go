package scan

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/andybalholm/cascadia"
	"golang.org/x/net/html"
)

// Field represents a struct field.
type field struct {
	index []int
	scan  ScanFunc
}

// StructScanner implements a struct scanner.
type StructScanner struct {
	fields []field
}

// Struct returns a scanfunc for a struct or an error.
func Struct(opts Options, t reflect.Type) (ScanFunc, error) {
	var fields []field

	for j := 0; j < t.NumField(); j++ {
		var f = t.Field(j)
		var tag = f.Tag
		var attr string
		var css string

		if f.PkgPath != "" {
			continue
		}

		if css = tag.Get("css"); len(css) == 0 {
			continue
		}

		if j := strings.IndexByte(css, '@'); j != -1 {
			attr = css[j+1:]
			css = css[:j]
		}

		sel, err := cascadia.Compile(css)
		if err != nil {
			return nil, err
		}

		scan, err := ScannerOf(f.Type, Options{
			Selector: sel,
			Attr:     attr,
		})
		if err != nil {
			return nil, err
		}

		fields = append(fields, field{
			index: f.Index,
			scan:  scan,
		})
	}

	if len(fields) == 0 {
		return nil, fmt.Errorf("scan: struct %v has no css tags", t)
	}

	return (StructScanner{fields}).scan, nil
}

// Scan implements a scan func.
func (ss StructScanner) scan(dst reflect.Value, src *html.Node) error {
	for _, f := range ss.fields {
		v := dst.FieldByIndex(f.index)
		if err := f.scan(v, src); err != nil {
			return err
		}
	}
	return nil
}

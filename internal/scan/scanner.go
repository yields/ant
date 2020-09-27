package scan

import (
	"fmt"
	"reflect"
	"sync"

	"golang.org/x/net/html"
)

// Scanner implements a type scanner.
//
// The scanner extracts raw HTML data into registered
// types. When the scanner sees a new type it will analyze
// it and create a scanfunc that can be re-used with different
// html nodes.
//
// The scanner supports the follow types:
//
//   - int
//   - uint
//   - float
//   - string
//   - byte
//   - struct
//
// The scanner also supports exporting multiple html nodes into
// a slice of the above types.
//
// The scanner uses tags to lookup data in the root html node
// and extract its data, for example:
//
//   AuthorURLs []string `css:"a.author@href"`
//
// Will export all hrefs from anchor tags that have the class `.author`.
type Scanner struct {
	types map[reflect.Type]ScanFunc
	mutex *sync.RWMutex
}

// NewScanner returns a new scanner.
func NewScanner() *Scanner {
	return &Scanner{
		types: make(map[reflect.Type]ScanFunc),
		mutex: &sync.RWMutex{},
	}
}

// Scan scans from src node into dst value.
func (s *Scanner) Scan(dst interface{}, src *html.Node, opts Options) error {
	var v = reflect.ValueOf(dst)
	var t = v.Type()

	if t.Kind() != reflect.Ptr {
		return fmt.Errorf("scan: cannot scan into non ptr %T", dst)
	}

	f, ok := s.lookup(t)
	if !ok {
		scanFunc, err := s.register(t.Elem(), opts)
		if err != nil {
			return err
		}
		f = scanFunc
	}

	return f(v.Elem(), src)
}

// Lookup returns a scan func from t.
func (s *Scanner) lookup(t reflect.Type) (ScanFunc, bool) {
	s.mutex.RLock()
	f, ok := s.types[t]
	s.mutex.RUnlock()
	return f, ok
}

// Register registers a scan func type.
func (s *Scanner) register(t reflect.Type, opts Options) (ScanFunc, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if f, ok := s.types[t]; ok {
		return f, nil
	}

	f, err := ScannerOf(t, opts)
	if err != nil {
		return nil, err
	}

	s.types[t] = f
	return f, nil
}

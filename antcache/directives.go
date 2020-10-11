package antcache

import (
	"net/http"
	"strconv"
	"strings"
	"time"
)

// Directives represents cache-control directives.
type directives map[string]string

// DirectivesFrom returns directives from h.
func directivesFrom(h http.Header) directives {
	var c = h.Get("Cache-Control")
	var d directives

	for _, item := range split(c, ",") {
		if d == nil {
			d = make(directives)
		}

		if j := strings.IndexByte(item, '='); j != -1 {
			d[item[:j]] = item[j+1:]
		} else {
			d[item] = ""
		}
	}

	return d
}

// Has returns true if directive name is set.
func (d directives) has(name string) bool {
	_, ok := d[name]
	return ok
}

// Duration parses directive duration by name.
//
// Ok is true if the directive exists and has a positive duration.
func (d directives) duration(name string) (td time.Duration, ok bool) {
	if v, has := d[name]; has {
		n, err := strconv.ParseInt(v, 10, 64)
		td, ok = (time.Duration(n) * time.Second), err == nil
	}
	return
}

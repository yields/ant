package ant

import (
	"sync"

	"github.com/andybalholm/cascadia"
)

var (
	// Selectors is a globally shared cache that is used
	// to compile selectors where they're needed.
	selectors = &selectorsCache{
		m:   make(map[string]cascadia.Selector),
		mtx: &sync.RWMutex{},
	}
)

// Selectors implements a selectors cache.
type selectorsCache struct {
	m   map[string]cascadia.Selector
	mtx *sync.RWMutex
}

// Compile compiles a selector.
//
// If the selector already exists in the cache
// the method returns the compiled selector.
func (s *selectorsCache) compile(sel string) cascadia.Selector {
	s.mtx.RLock()
	selector, ok := s.m[sel]
	s.mtx.RUnlock()

	if ok {
		return selector
	}

	selector, _ = cascadia.Compile(sel)

	s.mtx.Lock()
	s.m[sel] = selector
	s.mtx.Unlock()

	return selector
}

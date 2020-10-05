// Package selectors provides utilities to compile and cache CSS selectors.
//
// The package provides a cache implementation that caches selectors in-memory
// using a `sync.Map`, it also exposes the same methods to compile selectors
// using a global cache.
package selectors

import (
	"sync"

	"github.com/andybalholm/cascadia"
)

// Cache is a global cache of selectors.
var cache = NewCache()

// Compile compiles the given selector.
//
// It uses a global pre-initialized cache
// of selectors.
func Compile(selector string) (cascadia.Selector, error) {
	return cache.Compile(selector)
}

// Cache implementation.
type Cache struct {
	m sync.Map
}

// NewCache returns a new cache.
func NewCache() *Cache {
	return &Cache{}
}

// Compile compiles the given selector.
//
// The method returns an error if the selector is invalid
// subsequent calls return the cached selector.
func (c *Cache) Compile(selector string) (cascadia.Selector, error) {
	if s, ok := c.m.Load(selector); ok {
		return s.(cascadia.Selector), nil
	}

	v, err := cascadia.Compile(selector)
	if err != nil {
		return nil, err
	}

	c.m.Store(selector, v)
	return v, nil
}

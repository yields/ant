package ant

import (
	"context"
	"sync"

	"github.com/willf/bloom"
)

// Deduper represents a URL de-duplicator.
type Deduper interface {
	// Dedupe de-duplicates the given URLs.
	//
	// The method returns a new slice of URLs
	// that were not visited yet, it must be
	// thread-safe.
	//
	// The function is not required to normalize the URLs
	// the engine normalizes them before calling the method.
	//
	// If an error is returned that implements
	// `Temporary() bool` and returns true, the
	// engine will retry.
	Dedupe(ctx context.Context, urls URLs) (URLs, error)
}

// Dedupe implements an in-memory deduper.
type deduper struct {
	m *sync.Map
}

// DedupeMap returns a new deduper backed by sync.Map.
func DedupeMap() Deduper {
	return &deduper{new(sync.Map)}
}

// Dedupe implementation.
func (d *deduper) Dedupe(ctx context.Context, urls URLs) (URLs, error) {
	var ret = make(URLs, 0, len(urls))

	for _, u := range urls {
		if _, exists := d.m.LoadOrStore(u.String(), nil); !exists {
			ret = append(ret, u)
		}
	}

	return ret, nil
}

// Dedupebf implements a bloom filter deduper.
type dedupebf struct {
	filter *bloom.BloomFilter
}

// DedupeBF returns a new deduper backed by bloom filter.
func DedupeBF(k, m uint) Deduper {
	return &dedupebf{
		filter: bloom.New(k, m),
	}
}

// Dedupe implementation.
func (d *dedupebf) Dedupe(ctx context.Context, urls URLs) (URLs, error) {
	var ret = make(URLs, 0, len(urls))

	for _, u := range urls {
		v := []byte(u.String())
		if !d.filter.Test(v) {
			d.filter.Add(v)
			ret = append(ret, u)
		}
	}

	return ret, nil
}

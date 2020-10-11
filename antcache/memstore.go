package antcache

import (
	"context"
	"sync"
)

// Memstore implements an in-memory store.
type memstore struct {
	c sync.Map
}

// Store implementation.
func (m *memstore) Store(ctx context.Context, key uint64, value []byte) error {
	m.c.Store(key, value)
	return nil
}

// Load implementation.
func (m *memstore) Load(ctx context.Context, key uint64) ([]byte, error) {
	if v, ok := m.c.Load(key); ok {
		return v.([]byte), nil
	}
	return nil, nil
}

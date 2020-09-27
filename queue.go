package ant

import (
	"context"
	"io"
	"sync"
)

// Queue represents a URL queue.
//
// The queue must be thread-safe.
type Queue interface {
	// Enqueue enqueues the given set of URLs.
	//
	// The method returns an io.EOF if the queue was
	// closed and a context error if the context was
	// canceled.
	//
	// Any other error will be treated as a critical
	// error and will be porpagated.
	Enqueue(ctx context.Context, urls ...string) error

	// Dequeue dequeues a URL.
	//
	// The method returns a URL or io.EOF error if
	// the queue was stopped.
	//
	// The method blocks until a URL is available or
	// until the queue is closed.
	Dequeue(ctx context.Context) (string, error)

	// Close closes the queue.
	//
	// The method blocks until the queue is closed
	// any queued URLs are discarded.
	Close() error
}

// MemoryQueue implements a naive in-memory queue.
type memoryQueue struct {
	pending []string
	cond    *sync.Cond
	stopped bool
}

// MemoryQueue returns a new memory queue.
func MemoryQueue(size int) Queue {
	return &memoryQueue{
		pending: make([]string, 0, size),
		cond:    sync.NewCond(&sync.RWMutex{}),
		stopped: false,
	}
}

// Enqueue implementation.
func (mq *memoryQueue) Enqueue(ctx context.Context, urls ...string) error {
	if len(urls) == 0 {
		return nil
	}

	mq.cond.L.Lock()
	defer mq.cond.L.Unlock()

	if mq.stopped {
		return io.EOF
	}

	mq.pending = append(mq.pending, urls...)
	mq.cond.Broadcast()

	return nil
}

// Dequeue implementation.
func (mq *memoryQueue) Dequeue(ctx context.Context) (string, error) {
	mq.cond.L.Lock()
	defer mq.cond.L.Unlock()

	for len(mq.pending) == 0 && !mq.stopped {
		mq.cond.Wait()
	}

	if mq.stopped {
		return "", io.EOF
	}

	url := mq.pending[0]
	mq.pending = mq.pending[1:]

	return url, nil
}

// Close implementation.
func (mq *memoryQueue) Close() error {
	mq.cond.L.Lock()
	defer mq.cond.L.Unlock()

	mq.stopped = true
	mq.pending = mq.pending[:0]
	mq.cond.Broadcast()

	return nil
}

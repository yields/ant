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
	Enqueue(ctx context.Context, urls URLs) error

	// Dequeue dequeues a URL.
	//
	// The method returns a URL or io.EOF error if
	// the queue was stopped.
	//
	// The method blocks until a URL is available or
	// until the queue is closed.
	Dequeue(ctx context.Context) (*URL, error)

	// Done acknowledges a URL.
	//
	// When a URL has been handled by the engine the method
	// is called with the URL.
	Done(url *URL)

	// Wait blocks until the queue is closed.
	//
	// When the engine encounters an error, or there are
	// no more URLs to handle the method should unblock.
	Wait()

	// Close closes the queue.
	//
	// The method blocks until the queue is closed
	// any queued URLs are discarded.
	Close() error
}

// MemoryQueue implements a naive in-memory queue.
type memoryQueue struct {
	pending URLs
	cond    *sync.Cond
	stopped bool
	wg      *sync.WaitGroup
}

// MemoryQueue returns a new memory queue.
func MemoryQueue(size int) Queue {
	return &memoryQueue{
		pending: make(URLs, 0, size),
		cond:    sync.NewCond(&sync.RWMutex{}),
		stopped: false,
		wg:      &sync.WaitGroup{},
	}
}

// Enqueue implementation.
func (mq *memoryQueue) Enqueue(ctx context.Context, urls URLs) error {
	if len(urls) == 0 {
		return nil
	}

	mq.cond.L.Lock()
	defer mq.cond.L.Unlock()

	if mq.stopped {
		return io.EOF
	}

	if ctx.Err() != nil {
		return ctx.Err()
	}

	mq.pending = append(mq.pending, urls...)
	mq.wg.Add(len(urls))
	mq.cond.Broadcast()

	return nil
}

// Dequeue implementation.
func (mq *memoryQueue) Dequeue(ctx context.Context) (*URL, error) {
	mq.cond.L.Lock()
	defer mq.cond.L.Unlock()

	for len(mq.pending) == 0 && (!mq.stopped && ctx.Err() == nil) {
		mq.cond.Wait()
	}

	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	if mq.stopped {
		return nil, io.EOF
	}

	url := mq.pending[0]
	mq.pending = mq.pending[1:]

	return url, nil
}

// Done implementation.
func (mq *memoryQueue) Done(*URL) {
	mq.wg.Done()
}

// Wait implementation.
func (mq *memoryQueue) Wait() {
	mq.wg.Wait()
}

// Close implementation.
func (mq *memoryQueue) Close() error {
	mq.cond.L.Lock()
	defer mq.cond.L.Unlock()

	for range mq.pending {
		mq.wg.Done()
	}

	mq.stopped = true
	mq.pending = mq.pending[:0]
	mq.cond.Broadcast()

	return nil
}

package antcdp

import (
	"context"
	"fmt"

	"github.com/mafredri/cdp/devtool"
)

// Targets represents a pool of targets.
type targets struct {
	client  *devtool.DevTools
	targets chan *devtool.Target
}

// NewTargets returns a new targets pool with c.
func newTargets(c *devtool.DevTools) *targets {
	return &targets{
		client:  c,
		targets: make(chan *devtool.Target, 10),
	}
}

// Acquire attempts to acquire a target.
//
// The method blocks until a target is acquired, if the given context
// is canceled the method returns the context's error.
//
// If an error occures when a target is created the method returns the error.
func (t *targets) acquire(ctx context.Context) (*devtool.Target, error) {
	select {
	case v := <-t.targets:
		return v, nil

	case <-ctx.Done():
		return nil, ctx.Err()

	default:
		v, err := t.client.Create(ctx)
		if err != nil {
			return nil, fmt.Errorf("antcdp: create target - %w", err)
		}
		return v, nil
	}
}

// Release releases the given target.
func (t *targets) release(target *devtool.Target) error {
	select {
	case t.targets <- target:
	default:
	}
	return nil
}

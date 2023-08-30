package errgroupx

import (
	"context"
	"fmt"
	"runtime/debug"

	"golang.org/x/sync/errgroup"
)

// Group is a thin wrapper around golang.org/x/sync/errgroup.Group that helps not leak its context
// past the lifetime of the group.
type Group struct {
	inner   *errgroup.Group
	ctx     context.Context
	cancel  context.CancelFunc
	recover bool
}

// WithContext creates a Group as a child of the given context.
func WithContext(ctx context.Context) *Group {
	intermediateContext, cancel := context.WithCancel(ctx)
	g, groupContext := errgroup.WithContext(intermediateContext)

	return &Group{inner: g, ctx: groupContext, cancel: cancel}
}

// WithRecover sets up the group to recover panics from spawned goroutines.
// Recovered errors are returned as errors when awaiting the group.
func (g *Group) WithRecover() *Group {
	g.recover = true
	return g
}

// WithLimit calls SetLimit on the underlying errgroup.Group.
func (g *Group) WithLimit(n int) *Group {
	g.inner.SetLimit(n)
	return g
}

// Go launch the given function in a goroutine as a member of the group. If the function returns an
// error, the Group-scoped context will be canceled.
func (g *Group) Go(f func(ctx context.Context) error) {
	g.inner.Go(func() (err error) {
		defer func() {
			if !g.recover {
				return
			}
			if rec := recover(); rec != nil {
				err = fmt.Errorf("%s\n%s", rec, debug.Stack())
			}
		}()
		return f(g.ctx)
	})
}

// Wait for all child processes of the group to complete.
func (g *Group) Wait() error {
	return g.inner.Wait()
}

// Cancel the group, without waiting for it to exit.
func (g *Group) Cancel() {
	g.cancel()
}

// Close the group by canceling it and waiting for it.
func (g *Group) Close() error {
	g.cancel()
	return g.Wait()
}

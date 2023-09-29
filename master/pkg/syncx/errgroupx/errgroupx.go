package errgroupx

import (
	"context"

	"golang.org/x/sync/errgroup"
)

// Group is a thin wrapper around golang.org/x/sync/errgroup.Group that helps not leak its context
// past the lifetime of the group.
type Group struct {
	inner  *errgroup.Group
	ctx    context.Context
	cancel context.CancelFunc
}

// WithContext creates a Group as a child of the given context.
func WithContext(ctx context.Context) Group {
	intermediateContext, cancel := context.WithCancel(ctx)
	g, groupContext := errgroup.WithContext(intermediateContext)

	return Group{inner: g, ctx: groupContext, cancel: cancel}
}

// Context returns the context within this group.
func (g *Group) Context() context.Context {
	return g.ctx
}

// Go launch the given function in a goroutine as a member of the group. If the function returns an
// error, the Group-scoped context will be canceled.
func (g *Group) Go(f func(ctx context.Context) error) {
	g.inner.Go(func() error {
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

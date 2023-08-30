package waitgroupx

import (
	"context"

	"github.com/determined-ai/determined/master/pkg/syncx/errgroupx"
)

// Group is a thin wrapper around sync.WaitGroup that associates a cancelable context with it.
type Group struct {
	inner *errgroupx.Group
}

// WithContext creates a Group as a child of the given context.
func WithContext(ctx context.Context) Group {
	return Group{inner: errgroupx.WithContext(ctx)}
}

// Go launch the given function in a goroutine as a member of the group.
func (g *Group) Go(f func(ctx context.Context)) {
	g.inner.Go(func(ctx context.Context) error {
		f(ctx)
		return nil
	})
}

// Wait for all child processes of the group to complete.
func (g *Group) Wait() { _ = g.inner.Wait() }

// Cancel the group, without waiting for it to exit.
func (g *Group) Cancel() { g.inner.Cancel() }

// Close the group by canceling it and waiting for it.
func (g *Group) Close() { _ = g.inner.Close() }

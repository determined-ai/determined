package errgroupx

import (
	"context"

	"golang.org/x/sync/errgroup"
)

// Group represents an errGroup.
type Group struct {
	inner  *errgroup.Group
	ctx    context.Context
	cancel context.CancelFunc
}

// WithContext returns an errGroup with context.
func WithContext(ctx context.Context) *Group {
	ctx, cancel := context.WithCancel(ctx)
	g, ctx := errgroup.WithContext(ctx)
	return &Group{inner: g, ctx: ctx, cancel: cancel}
}

// SubGroup returns an Group with context.
func (g *Group) Subgroup() *Group {
	return WithContext(g.ctx)
}

// Go returns a goroutine.
func (g *Group) Go(f func(ctx context.Context) error) {
	g.inner.Go(func() error {
		return f(g.ctx)
	})
}

// Wait induces a wait.
func (g *Group) Wait() error {
	return g.inner.Wait()
}

// Cancel stops a process.
func (g *Group) Cancel() {
	g.cancel()
}

// Close cancels a process.
func (g *Group) Close() error {
	g.cancel()
	return g.Wait()
}

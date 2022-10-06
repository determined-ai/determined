package errgroupx

import (
	"context"

	"golang.org/x/sync/errgroup"
)

type Group struct {
	inner  *errgroup.Group
	ctx    context.Context
	cancel context.CancelFunc
}

func WithContext(ctx context.Context) *Group {
	ctx, cancel := context.WithCancel(ctx)
	g, ctx := errgroup.WithContext(ctx)
	return &Group{inner: g, ctx: ctx, cancel: cancel}
}

func (g *Group) Subgroup() *Group {
	return WithContext(g.ctx)
}

func (g *Group) Go(f func(ctx context.Context) error) {
	g.inner.Go(func() error {
		return f(g.ctx)
	})
}

func (g *Group) Wait() error {
	return g.inner.Wait()
}

func (g *Group) Cancel() {
	g.cancel()
}

func (g *Group) Close() error {
	g.cancel()
	return g.Wait()
}

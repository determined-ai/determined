package waitgroupx

import (
	"context"
	"sync"
)

// Group is a thin wrapper around sync.WaitGroup that associates a cancelable context with it.
type Group struct {
	inner  sync.WaitGroup
	ctx    context.Context
	cancel context.CancelFunc
}

// WithContext creates a Group as a child of the given context.
func WithContext(ctx context.Context) Group {
	ctx, cancel := context.WithCancel(ctx)
	return Group{ctx: ctx, cancel: cancel}
}

// Go launch the given function in a goroutine as a member of the group.
func (g *Group) Go(f func(ctx context.Context)) {
	g.inner.Add(1)
	go func() {
		defer g.inner.Done()
		f(g.ctx)
	}()
}

// Wait for all child processes of the group to complete.
func (g *Group) Wait() {
	g.inner.Wait()
}

// Cancel the group, without waiting for it to exit.
func (g *Group) Cancel() {
	g.cancel()
}

// Close the group by canceling it and waiting for it.
func (g *Group) Close() {
	g.cancel()
	g.Wait()
}

package actor

import (
	"container/list"
	"context"
	"sync"
)

type inbox struct {
	qLock      sync.Mutex
	queue      *list.List
	closed     bool
	queueEmpty *sync.Cond
}

func newInbox() *inbox {
	i := &inbox{queue: list.New()}
	i.queueEmpty = sync.NewCond(&i.qLock)
	return i
}

func (i *inbox) _add(ctx *Context) {
	i.queue.PushBack(ctx)
	i.queueEmpty.Signal()
}

func (i *inbox) tell(ctx context.Context, owner *Ref, sender *Ref, message Message) {
	i.qLock.Lock()
	defer i.qLock.Unlock()
	if i.closed {
		return
	}
	i._add(wrap(ctx, owner, sender, message, nil))
}

func (i *inbox) ask(ctx context.Context, owner *Ref, sender *Ref, message Message) Response {
	i.qLock.Lock()
	defer i.qLock.Unlock()
	if i.closed {
		return emptyResponse(sender)
	}
	resp := &response{source: owner, future: make(chan Message, 1)}
	i._add(wrap(ctx, owner, sender, message, resp.future))
	return resp
}

func (i *inbox) get() *Context {
	i.qLock.Lock()
	defer i.qLock.Unlock()
	if i.closed {
		return nil
	}

	for i.queue.Len() == 0 {
		i.queueEmpty.Wait()
	}

	return i.queue.Remove(i.queue.Front()).(*Context)
}

func (i *inbox) len() int {
	i.qLock.Lock()
	defer i.qLock.Unlock()
	return i.queue.Len()
}

func (i *inbox) close() {
	i.qLock.Lock()
	defer i.qLock.Unlock()
	i.closed = true
	for elem := i.queue.Front(); elem != nil; elem = elem.Next() {
		ctx := elem.Value.(*Context)
		if ctx.ExpectingResponse() {
			ctx.Respond(errNoResponse)
		}
	}
}

func wrap(ctx context.Context, r *Ref, sender *Ref, message Message, result chan Message) *Context {
	return &Context{inner: ctx, recipient: r, message: message, sender: sender, result: result}
}

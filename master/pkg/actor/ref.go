package actor

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"reflect"
	"runtime/debug"
	"strings"
	"sync"
	"time"

	"github.com/opentracing/opentracing-go"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

// Internal actor reference messages.
type (
	// stop is an internal message sent to actors to stop the actor.
	stop struct{}

	// createChild is an internal message sent to actors to create a child from the actor
	// implementation.
	createChild struct {
		address Address
		actor   Actor
	}

	// childCreated is an internal message sent as response when a child is created.
	childCreated struct {
		child   *Ref
		created bool
	}
)

// Ref is an immutable actor reference to an actor.
type Ref struct {
	log *log.Entry

	address        Address
	registeredTime time.Time

	system       *System
	actor        Actor
	parent       *Ref
	children     map[Address]*Ref
	deadChildren map[Address]bool
	inbox        *inbox

	// lLock locks on interactions with close listeners. When adding close listeners, if the actor
	// is already shut down, the error is returned. Otherwise, a new listener is created and will be
	// notified at the end of the actor shutdown process.
	lLock     sync.Mutex
	err       error
	listeners []chan error
	shutdown  bool

	tracing struct {
		tracer opentracing.Tracer
		closer io.Closer
	}
}

func newRef(system *System, parent *Ref, address Address, actor Actor) *Ref {
	typeName := reflect.TypeOf(actor).String()
	if strings.Contains(typeName, ".") {
		typeName = strings.Split(typeName, ".")[1]
	}
	ref := &Ref{
		log: log.WithField("type", typeName).WithField("id", address.Local()).WithField(
			"system", system.id),

		address:        address,
		registeredTime: time.Now(),

		system:       system,
		actor:        actor,
		parent:       parent,
		children:     make(map[Address]*Ref),
		deadChildren: make(map[Address]bool),
		inbox:        newInbox(),
	}

	if traceEnabled {
		addTracer(ref)
	}

	go ref.run()
	return ref
}

// Parent returns the reference to the actor's parent.
func (r *Ref) Parent() *Ref {
	return r.parent
}

// Children returns a list of references to the actor's children.
func (r *Ref) Children() []*Ref {
	children := make([]*Ref, 0, len(r.children))
	for _, child := range r.children {
		children = append(children, child)
	}
	return children
}

// Child returns the child with the given local ID.
func (r *Ref) Child(id interface{}) *Ref {
	return r.children[r.address.Child(id)]
}

// Address returns the address of the actor.
func (r *Ref) Address() Address {
	return r.address
}

// RegisteredTime returns the time that the actor registered with the system.
func (r *Ref) RegisteredTime() time.Time {
	return r.registeredTime
}

// System returns the underlying system that this actor belongs to.
func (r *Ref) System() *System {
	return r.system
}

func (r *Ref) String() string {
	return fmt.Sprintf("{%s (%T created at %v): %s://%s}",
		r.actor, r.actor, r.registeredTime, r.system.id, r.address.String())
}

func (r *Ref) tell(ctx context.Context, sender *Ref, message Message) {
	if traceEnabled {
		ctx = traceSend(ctx, sender, r, message, tellOperation)
	}
	r.inbox.tell(ctx, r, sender, message)
}

func (r *Ref) ask(ctx context.Context, sender *Ref, message Message) Response {
	if traceEnabled {
		ctx = traceSend(ctx, sender, r, message, askOperation)
	}
	return r.inbox.ask(ctx, r, sender, message)
}

// sendInternalMessage sends an actor framework message. These messages can be safely ignored by the
// actor.
func (r *Ref) sendInternalMessage(message Message) error {
	ctx := &Context{recipient: r, message: message}
	err := r.actor.Receive(ctx)
	// `errUnexpectedMessage` is ignored; other errors cause the actor to shut down.
	if _, ok := err.(errUnexpectedMessage); err != nil && !ok {
		return err
	}
	return nil
}

func (r *Ref) createChild(address Address, actor Actor) (*Ref, bool) {
	if existingRef, ok := r.children[address]; ok {
		return existingRef, false
	}

	ref := newRef(r.system, r, address, actor)
	r.children[address] = ref

	r.system.refsLock.Lock()
	defer r.system.refsLock.Unlock()

	r.system.refs[address] = ref

	return ref, true
}

func (r *Ref) createChildFromFactory(address Address, factory func() Actor) (*Ref, bool) {
	if existingRef, ok := r.children[address]; ok {
		return existingRef, false
	}

	ref := newRef(r.system, r, address, factory())
	r.children[address] = ref

	r.system.refsLock.Lock()
	defer r.system.refsLock.Unlock()

	r.system.refs[address] = ref

	return ref, true
}

func (r *Ref) deleteChild(address Address) {
	delete(r.children, address)

	r.system.refsLock.Lock()
	defer r.system.refsLock.Unlock()

	delete(r.system.refs, address)
}

func (r *Ref) processMessage() bool {
	ctx := r.inbox.get()

	r.log.Tracef("get %T, inbox length: %v", ctx.message, r.inbox.len())

	if traceEnabled {
		defer traceReceive(ctx, r)()
	}
	defer func() {
		if ctx.ExpectingResponse() {
			ctx.Respond(errNoResponse)
		}
	}()

	// Handle any internal state change messages first.
	switch typed := ctx.Message().(type) {
	case Ping:
		ctx.Respond(typed)
		return false
	case createChild:
		child, created := r.createChild(typed.address, typed.actor)
		ctx.Respond(childCreated{
			child:   child,
			created: created,
		})
		return false
	case ChildFailed:
		if _, ok := r.deadChildren[typed.Child.address]; ok {
			delete(r.deadChildren, typed.Child.address)
			return false
		}
		r.deleteChild(typed.Child.address)
		if r.err = r.sendInternalMessage(ctx.Message()); r.err != nil {
			return true
		}
		return false
	case ChildStopped:
		if _, ok := r.deadChildren[typed.Child.address]; ok {
			delete(r.deadChildren, typed.Child.address)
			return false
		}
		r.deleteChild(typed.Child.address)
		if r.err = r.sendInternalMessage(ctx.Message()); r.err != nil {
			return true
		}
		return false
	case stop:
		return true
	}

	// Any message not handled internally is sent to the actor implementation.
	if ctx.Sender() == nil || !r.deadChildren[ctx.Sender().address] {
		r.err = r.actor.Receive(ctx)
	}

	return r.err != nil
}

func (r *Ref) run() {
	defer r.close()
	if r.err = r.sendInternalMessage(PreStart{}); r.err != nil {
		return
	}
	for {
		if r.processMessage() {
			return
		}
	}
}

// Stop asynchronously notifies the actor to stop.
func (r *Ref) Stop() {
	r.tell(context.Background(), nil, stop{})
}

// AwaitTermination waits for the actor to stop, returning an error if the actor has failed during
// its lifecycle.
func (r *Ref) AwaitTermination() error {
	r.lLock.Lock()
	if r.shutdown {
		r.lLock.Unlock()
		return r.err
	}
	listener := make(chan error)
	r.listeners = append(r.listeners, listener)
	r.lLock.Unlock()
	return <-listener
}

// StopAndAwaitTermination synchronously stops the actor, returning an error if the actor fails to
// close properly.
func (r *Ref) StopAndAwaitTermination() error {
	r.Stop()
	return r.AwaitTermination()
}

func (r *Ref) close() {
	r.lLock.Lock()
	defer r.lLock.Unlock()

	if r.err != nil {
		r.log.WithError(r.err).Error("error while actor was running")
	}
	// Recover from an actor panic and set the error flag.
	if rec := recover(); rec != nil {
		r.log.Error(rec, "\n", string(debug.Stack()))
		r.err = errors.Errorf("unexpected panic: %v", rec)
	}

	// Drain the remaining messages in the inbox. All senders expecting results are sent an
	// ErrNoResponse.
	r.inbox.close()

	for _, child := range r.children {
		child.Stop()
	}

	// Stop and wait for all children to exit.
	for id, child := range r.children {
		if tErr := child.AwaitTermination(); tErr != nil {
			r.err = errors.Wrapf(tErr, "error closing child: %s", id)
		}
		r.deleteChild(r.address)
	}

	// Ask the underlying actor implementation to clean up.
	if err := r.sendInternalMessage(PostStop{}); err != nil {
		r.log.WithError(err).Error("error shutting down actor")
		if r.err == nil {
			r.err = err
		} else {
			r.err = errors.Wrap(r.err, err.Error())
		}
	}

	// Notify the parent that the actor is no longer processing messages.
	if r != r.system.Ref {
		if r.err != nil {
			r.parent.tell(context.Background(), r, ChildFailed{Child: r, Error: r.err})
		} else {
			r.parent.tell(context.Background(), r, ChildStopped{Child: r})
		}
	}

	// Notify all listeners that the actor has stopped.
	for _, listener := range r.listeners {
		if r.err != nil {
			listener <- r.err
		}
		close(listener)
	}

	// Close all resources used for tracing.
	if traceEnabled {
		closeTracer(r)
	}

	r.shutdown = true
}

// MarshalJSON implements the json.Marshaler interface.
func (r *Ref) MarshalJSON() ([]byte, error) {
	return json.Marshal(r.Address())
}

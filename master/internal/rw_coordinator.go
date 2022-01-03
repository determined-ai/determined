package internal

import (
	"fmt"

	"github.com/gorilla/websocket"

	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/actor/api"
)

// RWCoordinator-specific actor messages.
type resourceRequest struct {
	resource string
	readLock bool
	socket   *websocket.Conn
}

// Per resource metadata tracking lock usage.
type resourceRWLockStatus struct {
	readLockOwners    map[*actor.Ref]bool
	writeLockOwner    *actor.Ref
	readLocksWaiting  map[*actor.Ref]bool
	writeLocksWaiting map[*actor.Ref]bool
}

type rwCoordinator struct {
	resourceToLockStatusMap map[string]*resourceRWLockStatus
	actorToLockStatusMap    map[*actor.Ref]*resourceRWLockStatus
	numResourceRequests     int
}

func newRWCoordinator() actor.Actor {
	return &rwCoordinator{
		resourceToLockStatusMap: make(map[string]*resourceRWLockStatus),
		actorToLockStatusMap:    make(map[*actor.Ref]*resourceRWLockStatus),
		numResourceRequests:     0,
	}
}

func (r *rwCoordinator) Receive(ctx *actor.Context) error {
	switch msg := ctx.Message().(type) {
	case resourceRequest:
		if err := r.processResourceRequest(ctx, msg); err != nil {
			return err
		}
	case actor.ChildFailed:
		if err := r.processResourceRelease(ctx, msg.Child); err != nil {
			return err
		}
	case actor.ChildStopped:
		if err := r.processResourceRelease(ctx, msg.Child); err != nil {
			return err
		}
	default:
		break
	}
	return nil
}

func (r *rwCoordinator) processResourceRequest(
	ctx *actor.Context,
	msg resourceRequest,
) error {
	a := api.WrapSocket(msg.socket, nil, false)
	ref, _ := ctx.ActorOf(fmt.Sprintf("resourceRequest-socket-%d", r.numResourceRequests), a)
	// Create a unique identifier for every socket actor.
	r.numResourceRequests++
	ctx.Respond(ref)

	if _, ok := r.resourceToLockStatusMap[msg.resource]; !ok {
		r.resourceToLockStatusMap[msg.resource] = &resourceRWLockStatus{
			make(map[*actor.Ref]bool),
			nil,
			make(map[*actor.Ref]bool),
			make(map[*actor.Ref]bool),
		}
	}
	resource := r.resourceToLockStatusMap[msg.resource]
	r.actorToLockStatusMap[ref] = r.resourceToLockStatusMap[msg.resource]

	if msg.readLock {
		resource.readLocksWaiting[ref] = true
	} else {
		resource.writeLocksWaiting[ref] = true
	}

	if err := r.processWaitingRequests(ctx, resource); err != nil {
		return err
	}

	return nil
}

func (r *rwCoordinator) processWaitingRequests(
	ctx *actor.Context,
	resource *resourceRWLockStatus,
) error {
	if err := r.processWriteLockRequests(ctx, resource); err != nil {
		return err
	}

	if err := r.processReadLockRequests(ctx, resource); err != nil {
		return err
	}

	return nil
}

func (r *rwCoordinator) processReadLockRequests(
	ctx *actor.Context,
	resource *resourceRWLockStatus,
) error {
	if resource.writeLockOwner != nil || len(resource.writeLocksWaiting) != 0 {
		return nil
	}

	for ref := range resource.readLocksWaiting {
		delete(resource.readLocksWaiting, ref)
		resource.readLockOwners[ref] = true

		if err := api.WriteSocketRaw(ctx, ref, "read_lock_granted"); err != nil {
			ctx.Log().WithError(err).Errorf("cannot write to socket")
		}
	}

	return nil
}

func (r *rwCoordinator) processWriteLockRequests(
	ctx *actor.Context,
	resource *resourceRWLockStatus,
) error {
	if resource.writeLockOwner != nil && len(resource.readLockOwners) != 0 {
		return nil
	}

	for ref := range resource.writeLocksWaiting {
		delete(resource.writeLocksWaiting, ref)
		resource.writeLockOwner = ref

		if err := api.WriteSocketRaw(ctx, ref, "write_lock_granted"); err != nil {
			ctx.Log().WithError(err).Errorf("cannot write to socket")
		}

		// Can only grant one write lock at a time.
		break
	}

	return nil
}

func (r *rwCoordinator) processResourceRelease(ctx *actor.Context, ref *actor.Ref) error {
	if _, ok := r.actorToLockStatusMap[ref]; !ok {
		ctx.Log().Error("RW coordinator failed to release lock for unknown actor")
		return nil
	}

	resource := r.actorToLockStatusMap[ref]
	if ref == resource.writeLockOwner {
		resource.writeLockOwner = nil
	} else {
		delete(resource.readLockOwners, ref)
		delete(resource.readLocksWaiting, ref)
		delete(resource.writeLocksWaiting, ref)
	}
	delete(r.actorToLockStatusMap, ref)

	if err := r.processWaitingRequests(ctx, resource); err != nil {
		return err
	}

	return nil
}

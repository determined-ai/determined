package actors

import (
	"testing"

	"github.com/pkg/errors"
	"gotest.tools/assert"

	"github.com/determined-ai/determined/master/pkg/actor"
)

type ActorStopped struct{}

type listener struct {
	t           *testing.T
	notified    bool
	stoppingRef *actor.Ref
	expectedErr error
	closeErr    error
}

func (l *listener) Receive(context *actor.Context) error {
	switch msg := context.Message().(type) {
	case *actor.Ref:
		l.stoppingRef = msg
		NotifyOnStop(context, msg, ActorStopped{})
	case ActorStopped:
		l.notified = true
		context.Self().Stop()
	case actor.PostStop:
		return l.closeErr
	}
	return nil
}

func TestNotifyOnStop_BeforeStop(t *testing.T) {
	system := actor.NewSystem(t.Name())
	l := &listener{t: t}

	lRef, created := system.ActorOf(actor.Addr("listener"), l)
	assert.Assert(t, created)
	ref, created := system.ActorOf(actor.Addr("mock"), &listener{})
	assert.Assert(t, created)

	system.Tell(lRef, ref)

	assert.Assert(t, !l.notified)
	assert.NilError(t, ref.StopAndAwaitTermination())

	assert.NilError(t, lRef.AwaitTermination())
	assert.Assert(t, l.notified)
}

func TestNotifyOnStop_AfterStop(t *testing.T) {
	system := actor.NewSystem(t.Name())
	err := errors.New("TEST")
	l := &listener{t: t, expectedErr: err}

	lRef, created := system.ActorOf(actor.Addr("listener"), l)
	assert.Assert(t, created)
	ref, created := system.ActorOf(actor.Addr("mock"), &listener{closeErr: err})
	assert.Assert(t, created)

	assert.Assert(t, !l.notified)
	assert.Error(t, ref.StopAndAwaitTermination(), err.Error())

	system.Tell(lRef, ref)

	assert.NilError(t, lRef.AwaitTermination())
	assert.Assert(t, l.notified)
}

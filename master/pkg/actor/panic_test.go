package actor

import (
	"testing"

	"gotest.tools/assert"
)

type panicActor struct{}

func (panicActor) Receive(context *Context) error {
	if _, ok := context.Message().(PostStop); !ok {
		panic("forced panic")
	}
	return nil
}

func TestPanic(t *testing.T) {
	system := NewSystem(t.Name())
	ref, _ := system.ActorOf(Addr("test"), &panicActor{})
	assert.Error(t, ref.AwaitTermination(),
		"unexpected panic: forced panic")
}

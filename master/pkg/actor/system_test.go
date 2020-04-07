package actor

import (
	"testing"

	"gotest.tools/assert"
)

type mockActor struct {
	messages []Message
}

func (a *mockActor) Receive(context *Context) error {
	a.messages = append(a.messages, context.Message())
	if context.ExpectingResponse() {
		context.Respond(context.Message())
	}
	if err, ok := context.Message().(error); ok {
		return err
	}
	return nil
}

func TestNewSystem(t *testing.T) {
	system := NewSystem(t.Name())
	assert.NilError(t, system.StopAndAwaitTermination())
}

func TestSystem_AwaitTermination(t *testing.T) {
	system := NewSystem(t.Name())
	system.Stop()
	assert.NilError(t, system.AwaitTermination())
	assert.NilError(t, system.AwaitTermination())
}

func TestSystem_ActorOf(t *testing.T) {
	system := NewSystem(t.Name())
	ref1, created1 := system.ActorOf(Addr("mock"), &mockActor{})
	assert.Assert(t, ref1 != nil)
	assert.Assert(t, created1)

	ref2, created2 := system.ActorOf(Addr("mock"), &mockActor{})
	assert.Assert(t, ref2 == ref1)
	assert.Assert(t, !created2)

	ref3, created3 := system.ActorOf(Addr("nonexistent", "address"), &mockActor{})
	assert.Assert(t, ref3 == nil)
	assert.Assert(t, !created3)

	ref4, created4 := system.ActorOf(Addr("not root", "address"), &mockActor{})
	assert.Assert(t, ref4 == nil)
	assert.Assert(t, !created4)

	ref5, created5 := system.ActorOf(Addr("mock", "child"), &mockActor{})
	assert.Assert(t, ref5 != nil)
	assert.Assert(t, created5)

	ref6, created6 := system.ActorOf(Addr("mock", "child"), &mockActor{})
	assert.Assert(t, ref5 == ref6)
	assert.Assert(t, !created6)

	assert.NilError(t, system.StopAndAwaitTermination())
}

func TestSystem_AskAll(t *testing.T) {
	system := NewSystem(t.Name())
	ref1, _ := system.ActorOf(Addr("mock1"), &mockActor{})
	ref2, _ := system.ActorOf(Addr("mock1"), &mockActor{})
	ref3, _ := system.ActorOf(Addr("mock1"), &mockActor{})

	results := system.AskAll("result", ref1, ref2, ref3)
	assert.NilError(t, system.StopAndAwaitTermination())

	index := 0
	for result := range results {
		assert.Equal(t, result.Get(), "result")
		index++
	}
	assert.Equal(t, index, 3)
}

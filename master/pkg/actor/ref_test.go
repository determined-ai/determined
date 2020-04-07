package actor

import (
	"testing"
)

type panics struct{}

func (*panics) Receive(ctx *Context) error {
	switch ctx.Message().(type) {
	case string:
		panic("PANIC")
	}
	return nil
}

func TestRefResponseOnPanic(t *testing.T) {
	system := NewSystem("panic")
	ref, _ := system.ActorOf(Addr("panic"), &panics{})
	system.Ask(ref, "panic").Get()
}

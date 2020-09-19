package actor

import (
	"testing"
	"time"

	"gotest.tools/assert"
)

func TestResponseTimeout(t *testing.T) {
	system := NewSystem(t.Name())
	ref, _ := system.ActorOf(Addr("test"), ActorFunc(func(context *Context) error {
		if context.ExpectingResponse() {
			time.Sleep(1 * time.Second)
			context.Respond(false)
		}
		return nil
	}))
	result, ok := system.Ask(ref, "").GetOrElseTimeout(true, 1*time.Millisecond)
	assert.Assert(t, result.(bool))
	assert.Assert(t, !ok)
}

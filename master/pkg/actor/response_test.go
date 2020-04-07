package actor

import (
	"testing"
	"time"

	"gotest.tools/assert"
)

type noResponse struct {
}

func (*noResponse) Receive(context *Context) error {
	if context.ExpectingResponse() {
		time.Sleep(1 * time.Second)
		context.Respond(false)
	}
	return nil
}

func TestResponseTimeout(t *testing.T) {
	system := NewSystem(t.Name())
	ref, _ := system.ActorOf(Addr("test"), &noResponse{})
	result, ok := system.Ask(ref, "").GetOrElseTimeout(true, 1*time.Millisecond)
	assert.Assert(t, result.(bool))
	assert.Assert(t, !ok)
}

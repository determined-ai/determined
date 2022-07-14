package actor

import (
	"testing"
	"time"

	"gotest.tools/assert"
)

func TestResponseTimeout(t *testing.T) {
	system := NewSystem(t.Name())
	sleepDuration := 1 * time.Second
	ref, _ := system.ActorOf(Addr("test"), ActorFunc(func(context *Context) error {
		if context.ExpectingResponse() {
			time.Sleep(sleepDuration)
			context.Respond(false)
		}
		return nil
	}))
	start := time.Now()
	result, ok := system.Ask(ref, "").GetOrElseTimeout(true, 1*time.Millisecond)
	duration := time.Now().Sub(start)
	assert.Assert(t, duration < sleepDuration, "test duration: %d ms", duration.Milliseconds())
	assert.Assert(t, result.(bool))
	assert.Assert(t, !ok)
}

func TestResponseTimeoutOk(t *testing.T) {
	system := NewSystem(t.Name())
	ref, _ := system.ActorOf(Addr("test"), ActorFunc(func(context *Context) error {
		if context.ExpectingResponse() {
			context.Respond(false)
		}
		return nil
	}))
	start := time.Now()
	timeoutDuration := 1 * time.Second
	result, ok := system.Ask(ref, "").GetOrElseTimeout(true, timeoutDuration)
	duration := time.Now().Sub(start)
	assert.Assert(t, duration < timeoutDuration, "test duration: %d ms", duration.Milliseconds())
	assert.Assert(t, result.(bool) == false)
	assert.Assert(t, ok)
}

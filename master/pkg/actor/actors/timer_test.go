package actors

import (
	"testing"
	"time"

	"gotest.tools/assert"

	"github.com/determined-ai/determined/master/pkg/actor"
)

type timerTest struct {
	t     *testing.T
	wait  time.Duration
	msg   actor.Message
	start time.Time
}

type tick struct{}

func (a *timerTest) Receive(ctx *actor.Context) error {
	switch msg := ctx.Message().(type) {
	case tick:
		assert.Equal(a.t, msg, a.msg)
		assert.Assert(a.t, time.Now().After(a.start.Add(a.wait)))
		ctx.Self().System().Stop()
	case actor.PreStart:
		a.start = time.Now()
		NotifyAfter(ctx, a.wait, a.msg)
	}
	return nil
}

func TestNotifyAfter(t *testing.T) {
	system := actor.NewSystem(t.Name())
	system.ActorOf(actor.Addr("test"), &timerTest{t: t, wait: 300 * time.Millisecond, msg: tick{}})
	assert.NilError(t, system.AwaitTermination())
}

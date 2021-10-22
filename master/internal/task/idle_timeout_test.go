package task

import (
	"testing"
	"time"

	"gotest.tools/assert"

	"github.com/determined-ai/determined/master/pkg/actor"
)

type MockIdleTimeoutWatchee struct {
	idleTimeoutWatcher IdleTimeoutWatcher
}

func (m *MockIdleTimeoutWatchee) Receive(ctx *actor.Context) error {
	switch ctx.Message().(type) {
	case actor.PreStart:
		m.idleTimeoutWatcher.PreStart(ctx)
	case IdleTimeoutWatcherTick:
		return m.idleTimeoutWatcher.ReceiveMsg(ctx)
	case actor.PostStop:
	default:
		return actor.ErrUnexpectedMessage(ctx)
	}
	return nil
}

func TestIdleTimeoutWatcher(t *testing.T) {
	tickInterval := time.Millisecond
	lastActivity := time.Now()
	actionDone := false

	m := MockIdleTimeoutWatchee{
		idleTimeoutWatcher: IdleTimeoutWatcher{
			Timeout:              tickInterval,
			UseRunnerState:       true,
			lastExplicitActivity: &lastActivity,
			Action: func(ctx *actor.Context) {
				actionDone = true
			},
		},
	}

	system := actor.NewSystem(t.Name())
	mActor, created := system.ActorOf(actor.Addr("MockIdleTimeoutWatchee"), &m)
	assert.Assert(t, created)

	system.Ask(mActor, actor.Ping{}).Get()
	assert.Equal(t, actionDone, false)
	time.Sleep(2 * tickInterval)
	system.Ask(mActor, actor.Ping{}).Get()
	assert.Equal(t, actionDone, true)
}

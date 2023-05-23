package allocation

import (
	"testing"
	"time"

	"gotest.tools/assert"

	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/pkg/actor"
)

func TestIdleTimeoutWatcherUseRunnerState(t *testing.T) {
	system := actor.NewSystem(t.Name())

	var actionDone bool
	tickInterval := time.Second
	cfg := sproto.IdleTimeoutConfig{
		TimeoutDuration: tickInterval,
		UseRunnerState:  true,
	}
	w := NewIdleTimeoutWatcher("test", &cfg, system, func() {
		actionDone = true
	})
	defer w.Close()

	w.RecordActivity(time.Now())

	time.Sleep(tickInterval / 2)
	assert.Equal(t, actionDone, false)
	time.Sleep(tickInterval)
	assert.Equal(t, actionDone, true)
}

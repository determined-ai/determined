package scheduler

import (
	"testing"

	"gotest.tools/assert"

	"github.com/determined-ai/determined/master/pkg/actor"
)

func TestResourceProviderForwardMessage(t *testing.T) {
	system := actor.NewSystem(t.Name())
	defaultRP, created := system.ActorOf(actor.Addr("defaultRP"), NewDefaultRP(
		NewFairShareScheduler(),
		BestFit,
		nil,
		0,
	))
	assert.Assert(t, created)

	rpActor, created := system.ActorOf(actor.Addr("resourceProviders"),
		NewResourceProviders(defaultRP))
	assert.Assert(t, created)

	taskSummary := system.Ask(rpActor, GetTaskSummaries{}).Get()
	assert.DeepEqual(t, taskSummary, make(map[TaskID]TaskSummary))
	assert.NilError(t, rpActor.StopAndAwaitTermination())
}

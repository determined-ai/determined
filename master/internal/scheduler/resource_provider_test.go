package scheduler

import (
	"testing"

	"gotest.tools/assert"

	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/model"
)

func TestResourceProviderForwardMessage(t *testing.T) {
	system := actor.NewSystem(t.Name())
	rp := NewResourceProvider(
		"cluster",
		NewFairShareScheduler(),
		BestFit,
		nil,
		"/opt/determined",
		model.TaskContainerDefaultsConfig{},
		nil,
		0,
	)

	rpActor, created := system.ActorOf(actor.Addr("resourceProvider"), rp)
	assert.Assert(t, created)

	taskSummary := system.Ask(rpActor, GetTaskSummaries{}).Get()
	assert.DeepEqual(t, taskSummary, make(map[TaskID]TaskSummary))
	assert.NilError(t, rpActor.StopAndAwaitTermination())
}

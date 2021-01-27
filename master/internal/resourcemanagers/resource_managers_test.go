package resourcemanagers

import (
	"testing"

	"gotest.tools/assert"

	"github.com/determined-ai/determined/master/pkg/actor"
)

func TestResourceManagerForwardMessage(t *testing.T) {
	system := actor.NewSystem(t.Name())
	conf := &ResourceConfig{
		ResourceManager: &ResourceManagerConfig{
			AgentRM: &AgentResourceManagerConfig{
				Scheduler: &SchedulerConfig{
					FairShare:     &FairShareSchedulerConfig{},
					FittingPolicy: best,
				},
			},
		},
		ResourcePools: []ResourcePoolConfig{
			{
				PoolName:                 defaultResourcePoolName,
				MaxCPUContainersPerAgent: 100,
			},
		},
	}
	rpActor, created := system.ActorOf(actor.Addr("resourceManagers"),
		NewResourceManagers(system, conf, nil))
	assert.Assert(t, created)

	taskSummary := system.Ask(rpActor, GetTaskSummaries{}).Get()
	assert.DeepEqual(t, taskSummary, make(map[TaskID]TaskSummary))
	assert.NilError(t, rpActor.StopAndAwaitTermination())
}

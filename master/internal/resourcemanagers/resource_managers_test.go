package resourcemanagers

import (
	"testing"

	"github.com/determined-ai/determined/master/pkg/model"

	"github.com/labstack/echo/v4"

	"gotest.tools/assert"

	"github.com/determined-ai/determined/master/internal/sproto"
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
				MaxAuxContainersPerAgent: 100,
			},
		},
	}
	rpActor := Setup(system, echo.New(), conf, nil, nil)

	taskSummary := system.Ask(rpActor, sproto.GetTaskSummaries{}).Get()
	assert.DeepEqual(t, taskSummary, make(map[model.AllocationID]TaskSummary))
	assert.NilError(t, rpActor.StopAndAwaitTermination())
}

func TestResourceManagerValidateRPResourcesUnknown(t *testing.T) {
	// We can reliably run this check only for AWS, GCP, or Kube resource pools,
	// but initializing either of these in the test is not viable. So let's at least
	// check if we properly return "unknown" for on-prem-like setups.
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
				MaxAuxContainersPerAgent: 100,
			},
		},
	}

	Setup(system, echo.New(), conf, nil, nil)
	value, err := sproto.ValidateRPResources(system, defaultResourcePoolName, 1)
	assert.Assert(t, err == nil)
	assert.Assert(t, value)
}

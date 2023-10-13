package agentrm

import (
	"testing"

	"github.com/determined-ai/determined/master/pkg/model"

	"github.com/labstack/echo/v4"

	"gotest.tools/assert"

	"github.com/determined-ai/determined/master/internal/config"
	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/internal/user"
	"github.com/determined-ai/determined/master/pkg/actor"
)

const defaultResourcePoolName = "default"

func TestResourceManagerForwardMessage(t *testing.T) {
	user.InitService(nil, nil, nil)
	system := actor.NewSystem(t.Name())
	conf := &config.ResourceConfig{
		ResourceManager: &config.ResourceManagerConfig{
			AgentRM: &config.AgentResourceManagerConfig{
				Scheduler: &config.SchedulerConfig{
					FairShare:     &config.FairShareSchedulerConfig{},
					FittingPolicy: best,
				},
			},
		},
		ResourcePools: []config.ResourcePoolConfig{
			{
				PoolName:                 defaultResourcePoolName,
				MaxAuxContainersPerAgent: 100,
			},
		},
	}

	rm := New(system, nil, echo.New(), conf, nil, nil)

	taskSummary, err := rm.GetAllocationSummaries(sproto.GetAllocationSummaries{})
	assert.NilError(t, err)
	assert.DeepEqual(t, taskSummary, make(map[model.AllocationID]sproto.AllocationSummary))
	assert.NilError(t, rm.Ref().StopAndAwaitTermination())
}

func TestResourceManagerValidateRPResourcesUnknown(t *testing.T) {
	user.InitService(nil, nil, nil)
	// We can reliably run this check only for AWS, GCP, or Kube resource pools,
	// but initializing either of these in the test is not viable. So let's at least
	// check if we properly return "unknown" for on-prem-like setups.
	system := actor.NewSystem(t.Name())
	conf := &config.ResourceConfig{
		ResourceManager: &config.ResourceManagerConfig{
			AgentRM: &config.AgentResourceManagerConfig{
				Scheduler: &config.SchedulerConfig{
					FairShare:     &config.FairShareSchedulerConfig{},
					FittingPolicy: best,
				},
			},
		},
		ResourcePools: []config.ResourcePoolConfig{
			{
				PoolName:                 defaultResourcePoolName,
				MaxAuxContainersPerAgent: 100,
			},
		},
	}

	rm := New(system, nil, echo.New(), conf, nil, nil)

	resp, err := rm.ValidateCommandResources(sproto.ValidateCommandResourcesRequest{
		ResourcePool: defaultResourcePoolName,
		Slots:        1,
	})
	assert.Assert(t, err == nil, err)
	assert.Assert(t, resp.Fulfillable)
	assert.NilError(t, rm.Ref().StopAndAwaitTermination())
}

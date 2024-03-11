package agentrm

import (
	"testing"

	"github.com/determined-ai/determined/master/pkg/model"

	"github.com/labstack/echo/v4"

	"gotest.tools/assert"

	"github.com/determined-ai/determined/master/internal/config"
	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/internal/user"
)

const defaultResourcePoolName = "default"

func TestResourceManagerForwardMessage(t *testing.T) {
	user.InitService(nil, nil)
	conf := &config.ResourceConfig{
		RootManagerInternal: &config.ResourceManagerConfig{
			AgentRM: &config.AgentResourceManagerConfig{
				Scheduler: &config.SchedulerConfig{
					FairShare:     &config.FairShareSchedulerConfig{},
					FittingPolicy: best,
				},
			},
		},
		RootPoolsInternal: []config.ResourcePoolConfig{
			{
				PoolName:                 defaultResourcePoolName,
				MaxAuxContainersPerAgent: 100,
			},
		},
	}

	rm, err := New(nil, echo.New(), conf.ResourceManagers()[0], nil, nil)
	assert.NilError(t, err, "error initializing resource manager")

	taskSummary, err := rm.GetAllocationSummaries()
	assert.NilError(t, err)
	assert.DeepEqual(t, taskSummary, make(map[model.AllocationID]sproto.AllocationSummary))
	rm.stop()
}

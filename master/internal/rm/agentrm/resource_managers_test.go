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
	//nolint:exhaustruct
	conf := &config.ResourceConfig{
		ResourceManagers: config.ResourceManagersConfig{
			{
				AgentRM: &config.AgentResourceManagerConfigV1{
					Scheduler: &config.SchedulerConfig{
						FairShare:     &config.FairShareSchedulerConfig{},
						FittingPolicy: best,
					},
					ResourcePools: []config.ResourcePoolConfig{
						{
							PoolName:                 defaultResourcePoolName,
							MaxAuxContainersPerAgent: 100,
						},
					},
				},
			},
		},
	}

	rm := New(nil, echo.New(), conf, nil, nil)

	taskSummary, err := rm.GetAllocationSummaries(sproto.GetAllocationSummaries{})
	assert.NilError(t, err)
	assert.DeepEqual(t, taskSummary, make(map[model.AllocationID]sproto.AllocationSummary))
	rm.stop()
}

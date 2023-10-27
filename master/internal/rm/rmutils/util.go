package rmutils

import (
	"github.com/determined-ai/determined/master/internal/config"
	"github.com/determined-ai/determined/master/pkg/generatedproto/resourcepoolv1"
)

// ResourcePoolsToConfig converts proto objects to an internal resource pool config object.
func ResourcePoolsToConfig(pools []*resourcepoolv1.ResourcePool,
) []config.ResourcePoolConfig {
	rpConfigs := make([]config.ResourcePoolConfig, len(pools))
	for i, rp := range pools {
		rpConfigs[i] = config.ResourcePoolConfig{
			PoolName: rp.Name,
		}
	}

	return rpConfigs
}

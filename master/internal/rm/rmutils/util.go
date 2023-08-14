package rmutils

import (
	"github.com/determined-ai/determined/master/internal/config"
	"github.com/determined-ai/determined/proto/pkg/resourcepoolv1"
)

// ResourcePoolsToConfig converts proto objects to an internal resource pool config object.
func ResourcePoolsToConfig(pools []*resourcepoolv1.ResourcePool,
) ([]config.ResourcePoolConfig, error) {
	rpConfigs := make([]config.ResourcePoolConfig, len(pools))
	for i, rp := range pools {
		rpConfigs[i] = config.ResourcePoolConfig{
			PoolName: rp.Name,
		}
	}

	return rpConfigs, nil
}

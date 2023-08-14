package rmutils

import (
	"github.com/determined-ai/determined/master/internal/config"
	"github.com/determined-ai/determined/proto/pkg/resourcepoolv1"
)

// ResourcePoolsToConfig converts proto objects to an internal resource pool config object.
func ResourcePoolsToConfig(pools []*resourcepoolv1.ResourcePool,
) ([]config.ResourcePoolConfig, error) {
	var rpConfigs []config.ResourcePoolConfig
	for _, rp := range pools {
		rpConfigs = append(rpConfigs, config.ResourcePoolConfig{
			PoolName: rp.Name,
		})
	}

	return rpConfigs, nil
}

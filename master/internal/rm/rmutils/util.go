package rmutils

import (
	"github.com/determined-ai/determined/master/internal/config"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
)

// GetResourcePoolsResponseToConfig converts the proto message to an internal
// resource pool config object.
func GetResourcePoolsResponseToConfig(resp *apiv1.GetResourcePoolsResponse,
) ([]config.ResourcePoolConfig, error) {
	if resp == nil {
		return []config.ResourcePoolConfig{}, nil
	}

	var rpConfigs []config.ResourcePoolConfig
	for _, rp := range resp.ResourcePools {
		rpConfigs = append(rpConfigs, config.ResourcePoolConfig{
			PoolName: rp.Name,
		})
	}

	return rpConfigs, nil
}

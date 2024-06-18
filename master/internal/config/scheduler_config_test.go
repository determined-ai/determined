package config

import (
	"testing"

	"github.com/ghodss/yaml"
	"github.com/stretchr/testify/require"
)

func TestResourcePoolDefaults(t *testing.T) {
	resourcePoolDefault := dbConfig + `
resource_manager:
  type: agent
  scheduler:
    type: priority
    fitting_policy: best

resource_pools:
- pool_name: default
  scheduler:
    fitting_policy: worst
`
	unmarshaled := Config{}
	err := yaml.Unmarshal([]byte(resourcePoolDefault), &unmarshaled, yaml.DisallowUnknownFields)
	require.NoError(t, err)
	require.NoError(t, unmarshaled.Resolve())

	rm := unmarshaled.ResourceManagers()
	require.Len(t, rm, 1)
	rp := rm[0].ResourcePools
	require.Len(t, rp, 1)

	require.Equal(t, PriorityScheduling, rm[0].ResourceManager.AgentRM.Scheduler.GetType())
	require.Equal(t, PriorityScheduling, rp[0].Scheduler.GetType())
}

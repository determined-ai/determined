package config

import (
	"testing"

	"github.com/ghodss/yaml"
	"github.com/stretchr/testify/require"
)

func TestResourcePoolDefaults(t *testing.T) {
	dbConfig := `
db:
  user: config_file_user
  password: password
  host: hostname
  port: "3000"`

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
	require.Equal(t, 1, len(rm))
	rp := rm[0].ResourcePools
	require.Equal(t, 1, len(rp))

	require.Equal(t, PriorityScheduling, rm[0].ResourceManager.AgentRM.Scheduler.GetType())
	require.Equal(t, PriorityScheduling, rp[0].Scheduler.GetType())
}

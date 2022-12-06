//nolint:exhaustivestruct
package kubernetesrm

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/determined-ai/determined/master/pkg/device"
	"github.com/determined-ai/determined/master/pkg/schemas/expconf"

	k8sV1 "k8s.io/api/core/v1"
)

func TestLaterEnvironmentVariablesGetSet(t *testing.T) {
	dontBe := k8sV1.EnvVar{Name: "var", Value: "dontbe"}
	shouldBe := k8sV1.EnvVar{Name: "var", Value: "shouldbe"}
	env := expconf.EnvironmentConfig{
		RawEnvironmentVariables: &expconf.EnvironmentVariablesMap{
			RawCPU: []string{
				dontBe.Name + "=" + dontBe.Value,
				shouldBe.Name + "=" + shouldBe.Value,
			},
		},
	}

	p := pod{}
	actual, err := p.configureEnvVars(make(map[string]string), env, device.CPU)
	require.NoError(t, err)
	require.NotContains(t, actual, dontBe, "earlier variable set")
	require.Contains(t, actual, shouldBe, "later variable not set")
}

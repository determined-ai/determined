//nolint:exhaustivestruct
package kubernetesrm

import (
	"testing"
	"unicode"

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

func TestAllPrintableCharactersInEnv(t *testing.T) {
	expectedValue := ""
	for i := 0; i <= 1024; i++ {
		if unicode.IsPrint(rune(i)) {
			expectedValue += string([]rune{rune(i)})
		}
	}

	env := expconf.EnvironmentConfig{
		RawEnvironmentVariables: &expconf.EnvironmentVariablesMap{
			RawCPU: []string{
				"test=" + expectedValue,
				"test2",
				"func=f(x)=x",
			},
		},
	}

	p := pod{}
	actual, err := p.configureEnvVars(make(map[string]string), env, device.CPU)
	require.NoError(t, err)
	require.Contains(t, actual, k8sV1.EnvVar{Name: "test", Value: expectedValue})
	require.Contains(t, actual, k8sV1.EnvVar{Name: "test2", Value: ""})
	require.Contains(t, actual, k8sV1.EnvVar{Name: "func", Value: "f(x)=x"})
}

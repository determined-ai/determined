package dockerflags

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
)

func TestParseDockerFlags(t *testing.T) {
	conf, hostConf, networkConf, err := Parse([]string{"--cpu-shares", "1025"})
	require.NoError(t, err)
	require.NotNil(t, conf)
	require.Equal(t, int64(1025), hostConf.CPUShares)
	require.NotNil(t, networkConf)

	conf, hostConf, networkConf, err = Parse([]string{
		"--mac-address", "00:00:5e:00:53:af",
	})
	require.NoError(t, err)
	require.Equal(t, "00:00:5e:00:53:af", conf.MacAddress)
	require.NotNil(t, hostConf)
	require.NotNil(t, networkConf)

	_, _, _, err = Parse([]string{"--this-isnt-a-docker-flag", "false"}) // nolint: dogsled
	require.Error(t, err)

	_, _, _, err = Parse([]string{}) // nolint: dogsled
	require.NoError(t, err)

	// Set a value to a default value and ensure we get the same as initializing the
	// config struct through just doing container.Config{}.
	conf, hostConf, networkConf, err = Parse([]string{"--shm-size", "0"})
	require.NoError(t, err)
	require.Equal(t, &container.Config{}, conf)
	require.Equal(t, &container.HostConfig{}, hostConf)
	require.Equal(t, &network.NetworkingConfig{}, networkConf)
}

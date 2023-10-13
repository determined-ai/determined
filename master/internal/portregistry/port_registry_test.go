package portregistry

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/determined-ai/determined/master/internal/config"
)

var (
	dtrainSSHPortBase              = 12350
	interTrainProcessCommPort1Base = 12360
	interTrainProcessCommPort2Base = 12365
	c10DPortBase                   = 29400
)

func TestPortportRegistry(t *testing.T) {
	InitPortRegistry()
	port, err := GetPort(dtrainSSHPortBase)
	require.NoError(t, err)
	require.Equal(t, 12350, port)
	port, err = GetPort(dtrainSSHPortBase)
	require.NoError(t, err)
	require.Equal(t, 12351, port)
	port, err = GetPort(dtrainSSHPortBase)
	require.NoError(t, err)
	require.Equal(t, 12352, port)
	ReleasePort(12351)
	port, err = GetPort(dtrainSSHPortBase)
	require.NoError(t, err)
	require.Equal(t, 12351, port)
	port, err = GetPort(c10DPortBase)
	require.NoError(t, err)
	require.Equal(t, 29400, port)
	port, err = GetPort(c10DPortBase)
	require.NoError(t, err)
	require.Equal(t, 29401, port)
	ReleasePort(12350)
	ReleasePort(12351)
	port, err = GetPort(dtrainSSHPortBase)
	require.NoError(t, err)
	require.Equal(t, 12350, port)
	port, err = GetPort(c10DPortBase)
	require.NoError(t, err)
	require.Equal(t, 29402, port)
	port, err = GetPort(c10DPortBase)
	require.NoError(t, err)
	require.Equal(t, 29403, port)
	port, err = GetPort(interTrainProcessCommPort1Base)
	require.NoError(t, err)
	require.Equal(t, 12360, port)
	port, err = GetPort(dtrainSSHPortBase)
	require.NoError(t, err)
	require.Equal(t, 12351, port)
	port, err = GetPort(dtrainSSHPortBase)
	require.NoError(t, err)
	require.Equal(t, 12353, port)
	port, err = GetPort(dtrainSSHPortBase)
	require.NoError(t, err)
	require.Equal(t, 12354, port)
	port, err = GetPort(dtrainSSHPortBase)
	require.NoError(t, err)
	require.Equal(t, 12355, port)
	port, err = GetPort(dtrainSSHPortBase)
	require.NoError(t, err)
	require.Equal(t, 12356, port)
	port, err = GetPort(dtrainSSHPortBase)
	require.NoError(t, err)
	require.Equal(t, 12357, port)
	port, err = GetPort(dtrainSSHPortBase)
	require.NoError(t, err)
	require.Equal(t, 12358, port)
	port, err = GetPort(dtrainSSHPortBase)
	require.NoError(t, err)
	require.Equal(t, 12359, port)
	port, err = GetPort(dtrainSSHPortBase)
	require.NoError(t, err)
	require.Equal(t, 12361, port)
	port, err = GetPort(interTrainProcessCommPort1Base)
	require.NoError(t, err)
	require.Equal(t, 12362, port)
	port, err = GetPort(interTrainProcessCommPort2Base)
	require.NoError(t, err)
	require.Equal(t, 12365, port)
	port, err = GetPort(dtrainSSHPortBase)
	require.NoError(t, err)
	require.Equal(t, 12363, port)
	ReleasePort(12363)
	RestorePort(12363)
	port, err = GetPort(dtrainSSHPortBase)
	require.NoError(t, err)
	require.Equal(t, 12364, port)
	port, err = GetPort(interTrainProcessCommPort2Base)
	require.NoError(t, err)
	require.Equal(t, 12366, port)
	ReleasePort(12365)
	port, err = GetPort(interTrainProcessCommPort2Base)
	require.NoError(t, err)
	require.Equal(t, 12365, port)
}

func TestReservedPorts(t *testing.T) {
	config.GetMasterConfig().ReservedPorts = []int{dtrainSSHPortBase}
	InitPortRegistry()
	port, err := GetPort(dtrainSSHPortBase)
	require.NoError(t, err)
	require.Equal(t, dtrainSSHPortBase+1, port, "default port reserved; expect next highest")
}

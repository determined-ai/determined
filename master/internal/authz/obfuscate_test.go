package authz

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/determined-ai/determined/proto/pkg/containerv1"
	"github.com/determined-ai/determined/proto/pkg/devicev1"
)

func assertContainerDeviceObfuscated(t *testing.T, device *devicev1.Device) {
	devBrand := device.Brand
	devType := device.Type
	require.Equal(t, int32(hiddenInt), device.Id)
	require.Equal(t, hiddenString, device.Uuid)
	require.Equal(t, devBrand, device.Brand)
	require.Equal(t, devType, device.Type)
}

func TestObfuscateContainer(t *testing.T) {
	dev1 := devicev1.Device{
		Id:    int32(-101),
		Brand: "devBrand1",
		Uuid:  "devUuid1",
		Type:  devicev1.Type_TYPE_CPU,
	}
	dev2 := devicev1.Device{
		Id:    int32(-102),
		Brand: "devBrand2",
		Uuid:  "devUuid2",
		Type:  devicev1.Type_TYPE_CUDA,
	}
	container := containerv1.Container{
		Id:      "contID",
		Parent:  "parentID",
		State:   containerv1.State_STATE_RUNNING,
		Devices: []*devicev1.Device{&dev1, &dev2},
	}

	contState := container.State
	err := ObfuscateContainer(&container)
	require.NoError(t, err)
	require.Equal(t, hiddenString, container.Id)
	require.Equal(t, hiddenString, container.Parent)
	require.Equal(t, contState, container.State)
	require.True(t, container.PermissionDenied)
	for _, device := range container.Devices {
		assertContainerDeviceObfuscated(t, device)
	}
}

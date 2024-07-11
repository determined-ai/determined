package authz

import (
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/determined-ai/determined/proto/pkg/agentv1"
	"github.com/determined-ai/determined/proto/pkg/containerv1"
	"github.com/determined-ai/determined/proto/pkg/devicev1"
	"github.com/determined-ai/determined/proto/pkg/experimentv1"
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

func TestObfuscateAgentSlots(t *testing.T) {
	// Loop test so we know that we aren't relying on any random chances.
	for i := 0; i < 100; i++ {
		agent := &agentv1.Agent{
			Slots: map[string]*agentv1.Slot{
				"005": {
					Id: "005",
					Container: &containerv1.Container{
						Id:     "contID",
						Parent: "parentID",
						State:  containerv1.State_STATE_RUNNING,
					},
					Device: &devicev1.Device{},
				},
				"006": {
					Id:     "006",
					Device: &devicev1.Device{},
				},
			},
		}

		require.NoError(t, ObfuscateAgent(agent))
		require.NotNil(t, agent.Slots["000"].Container)
		require.Nil(t, agent.Slots["001"].Container)
	}
}

func TestObfuscateExperiments(t *testing.T) {
	mustMakeStruct := func(v map[string]interface{}) *structpb.Struct {
		p, err := structpb.NewStruct(v)
		require.NoError(t, err)
		return p
	}
	tests := [...]struct {
		name           string
		config         *structpb.Struct
		expectedConfig map[string]interface{}
	}{
		{
			name:           "no data defined",
			config:         mustMakeStruct(map[string]interface{}{}),
			expectedConfig: map[string]interface{}{},
		},
		{
			name: "no secrets defined",
			config: mustMakeStruct(map[string]interface{}{
				"data": map[string]interface{}{
					"public_values": map[string]interface{}{
						"key1": "baef4876-7ff8-4aea-a022-9480606cb467",
					},
				},
			}),
			expectedConfig: map[string]interface{}{
				"data": map[string]interface{}{
					"public_values": map[string]interface{}{
						"key1": "baef4876-7ff8-4aea-a022-9480606cb467",
					},
				},
			},
		},
		{
			name: "some secrets defined",
			config: mustMakeStruct(map[string]interface{}{
				"data": map[string]interface{}{
					"public_values": map[string]interface{}{
						"key2": "03d43c5b-d227-433d-aee6-0121500ac0bb",
					},
					"secrets": map[string]interface{}{
						"key3": "58cb0887-c717-4b63-b274-2656f2fc4f2d",
						"key4": "7bba99b0-0227-4565-834d-8ca547c309f6",
					},
				},
			}),
			expectedConfig: map[string]interface{}{
				"data": map[string]interface{}{
					"public_values": map[string]interface{}{
						"key2": "03d43c5b-d227-433d-aee6-0121500ac0bb",
					},
					"secrets": map[string]interface{}{
						"key3": hiddenString,
						"key4": hiddenString,
					},
				},
			},
		},
	}

	for _, tt := range tests {
		config, expectedConfig := tt.config, tt.expectedConfig
		t.Run(tt.name, func(t *testing.T) {
			exp := experimentv1.Experiment{
				Config: config,
			}
			ObfuscateExperiments(&exp)
			require.Equal(t, expectedConfig, exp.Config.AsMap()) //nolint:staticcheck
		})
	}
}

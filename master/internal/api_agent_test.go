package internal

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/determined-ai/determined/proto/pkg/agentv1"
	"github.com/determined-ai/determined/proto/pkg/containerv1"
	"github.com/determined-ai/determined/proto/pkg/devicev1"
)

func TestSummarizeSlots_EmptySlots(t *testing.T) {
	slots := make(map[string]*agentv1.Slot)
	stats := SummarizeSlots(slots)

	assert.Equal(t, 0, len(stats.TypeStats))
	assert.Equal(t, 0, len(stats.BrandStats))
}

func TestSummarizeSlots_VariousStates(t *testing.T) {
	slots := map[string]*agentv1.Slot{
		"slot1": {
			Device: &devicev1.Device{
				Type:  devicev1.Type_TYPE_CUDA,
				Brand: "Nvidia",
			},
			Enabled:   true,
			Draining:  false,
			Container: &containerv1.Container{State: containerv1.State_STATE_RUNNING},
		},
		"slot2": {
			Device: &devicev1.Device{
				Type:  devicev1.Type_TYPE_CUDA,
				Brand: "Nvidia",
			},
			Enabled:  false,
			Draining: false,
		},
		"slot3": {
			Device: &devicev1.Device{
				Type:  devicev1.Type_TYPE_CPU,
				Brand: "Intel",
			},
			Enabled:  true,
			Draining: true,
		},
	}

	stats := SummarizeSlots(slots)

	assert.Equal(t, 2, int(stats.TypeStats[devicev1.Type_TYPE_CUDA.String()].Total))
	assert.Equal(t, 1, int(stats.TypeStats[devicev1.Type_TYPE_CPU.String()].Total))
	assert.Equal(t, 1, int(stats.TypeStats[devicev1.Type_TYPE_CUDA.String()].Disabled))
	assert.Equal(t, 1, int(stats.TypeStats[devicev1.Type_TYPE_CPU.String()].Draining))
	assert.Equal(t, 1, int(stats.TypeStats[devicev1.Type_TYPE_CUDA.String()].States[containerv1.State_STATE_RUNNING.String()]))

	assert.Equal(t, 2, int(stats.BrandStats["Nvidia"].Total))
	assert.Equal(t, 1, int(stats.BrandStats["Intel"].Total))
	assert.Equal(t, 1, int(stats.BrandStats["Nvidia"].Disabled))
	assert.Equal(t, 1, int(stats.BrandStats["Intel"].Draining))
}

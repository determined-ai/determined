package workload

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"gotest.tools/assert"
)

func roundTrip(t *testing.T, original, shell interface{}) interface{} {
	blob, err := json.Marshal(original)
	assert.NilError(t, err)

	err = json.Unmarshal(blob, shell)
	assert.NilError(t, err)
	return shell
}

func TestCompletedMessageMarshaling(t *testing.T) {
	original := &CompletedMessage{
		Type: "WORKLOAD_COMPLETED",
		Workload: Workload{
			Kind:         CheckpointModel,
			ExperimentID: 1,
			TrialID:      2,
			StepID:       3,
		},
		RawMetrics:        []byte("{}"),
		CheckpointMetrics: &CheckpointMetrics{},
		StartTime:         time.Now().Round(0),
		EndTime:           time.Now().Round(0),
	}
	rebuilt := roundTrip(t, original, &CompletedMessage{}).(*CompletedMessage)
	assert.DeepEqual(t, *original, *rebuilt)
}

func TestCompletedCheckpointMarshaling(t *testing.T) {
	metrics := &CheckpointMetrics{
		UUID: uuid.New(),
		Resources: map[string]int{
			"resource": 100,
		},
	}
	rawMetrics, err := json.Marshal(metrics)
	assert.NilError(t, err)
	original := &CompletedMessage{
		Type: "WORKLOAD_COMPLETED",
		Workload: Workload{
			Kind:         CheckpointModel,
			ExperimentID: 1,
			TrialID:      2,
			StepID:       3,
		},
		RawMetrics: rawMetrics,
		StartTime:  time.Now().Round(0),
		EndTime:    time.Now().Round(0),
	}
	rebuilt := roundTrip(t, original, &CompletedMessage{}).(*CompletedMessage)
	assert.DeepEqual(t, metrics, rebuilt.CheckpointMetrics)
	assert.Assert(t, rebuilt.RunMetrics == nil)
	assert.Assert(t, rebuilt.ValidationMetrics == nil)
}

func TestCompletedValidationMarshaling(t *testing.T) {
	metrics := &ValidationMetrics{
		NumInputs: 1,
		Metrics: map[string]interface{}{
			"metric": 2.0,
		},
	}
	rawMetrics, err := json.Marshal(metrics)
	assert.NilError(t, err)
	original := &CompletedMessage{
		Type: "WORKLOAD_COMPLETED",
		Workload: Workload{
			Kind:         ComputeValidationMetrics,
			ExperimentID: 1,
			TrialID:      2,
			StepID:       3,
		},
		RawMetrics: rawMetrics,
		StartTime:  time.Now().Round(0),
		EndTime:    time.Now().Round(0),
	}
	rebuilt := roundTrip(t, original, &CompletedMessage{}).(*CompletedMessage)
	assert.DeepEqual(t, metrics, rebuilt.ValidationMetrics)
	assert.Assert(t, rebuilt.RunMetrics == nil)
	assert.Assert(t, rebuilt.CheckpointMetrics == nil)
}

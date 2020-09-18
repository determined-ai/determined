package workload

import (
	"encoding/json"
	"testing"

	"gotest.tools/assert"
)

func TestWorkloadMarshaling(t *testing.T) {
	marshaled := Workload{
		Kind:                  RunStep,
		ExperimentID:          1,
		TrialID:               2,
		StepID:                3,
		NumBatches:            10,
		TotalBatchesProcessed: 0,
	}
	blob, marshalErr := json.Marshal(marshaled)
	assert.NilError(t, marshalErr)

	unmarshaled := Workload{}
	unmarshalErr := json.Unmarshal(blob, &unmarshaled)
	assert.NilError(t, unmarshalErr)
	assert.DeepEqual(t, marshaled, unmarshaled)
}

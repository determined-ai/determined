package expconf

import (
	"encoding/json"
	"testing"

	"gotest.tools/assert"
)

func TestNestedHPs(t *testing.T) {
	hp := make(map[string]HyperparameterV0)
	hp["learning_rate"] = HyperparameterV0{
		RawConstHyperparameter: &ConstHyperparameterV0{RawVal: 0.01},
	}
	hps := HyperparametersV0{
		"optimizer": HyperparameterV0{
			RawNestedHyperparameter: &hp,
		},
	}

	// Test marshaling.
	bytes, err := json.Marshal(hps)
	assert.NilError(t, err)

	var rawObj HyperparametersV0
	err = json.Unmarshal(bytes, &rawObj)
	assert.NilError(t, err)

	assert.DeepEqual(t, rawObj, hps)
}

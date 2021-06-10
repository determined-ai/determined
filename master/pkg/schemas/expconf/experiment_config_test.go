package expconf

import (
	"encoding/json"
	"testing"

	"gotest.tools/assert"

	"github.com/determined-ai/determined/master/pkg/ptrs"
)

func TestName(t *testing.T) {
	config := ExperimentConfig{
		RawName: Name{
			RawString: ptrs.StringPtr("my_name"),
		},
	}

	// Test marshaling.
	bytes, err := json.Marshal(config)
	assert.NilError(t, err)

	rawObj := map[string]interface{}{}
	err = json.Unmarshal(bytes, &rawObj)
	assert.NilError(t, err)

	var expect interface{} = "my_name"
	assert.DeepEqual(t, rawObj["name"], expect)

	// Test unmarshaling.
	newConfig := ExperimentConfig{}
	err = json.Unmarshal(bytes, &newConfig)
	assert.NilError(t, err)

	assert.DeepEqual(t, newConfig.Name().String(), "my_name")
}

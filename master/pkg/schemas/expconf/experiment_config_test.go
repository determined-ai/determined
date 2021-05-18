package expconf

import (
	"encoding/json"
	"testing"

	"gotest.tools/assert"

	"github.com/determined-ai/determined/master/pkg/ptrs"
	"github.com/determined-ai/determined/master/pkg/schemas"
)

func TestBindMountsMerge(t *testing.T) {
	e1 := ExperimentConfig{
		RawBindMounts: BindMountsConfig{
			BindMount{
				RawHostPath:      "/host/e1",
				RawContainerPath: "/container/e1",
			},
		},
	}
	e2 := ExperimentConfig{
		RawBindMounts: BindMountsConfig{
			BindMount{
				RawHostPath:      "/host/e2",
				RawContainerPath: "/container/e2",
			},
		},
	}
	out := schemas.Merge(e1, e2).(ExperimentConfig)
	assert.Assert(t, len(out.RawBindMounts) == 2)
	assert.Assert(t, out.RawBindMounts[0].RawHostPath == "/host/e1")
	assert.Assert(t, out.RawBindMounts[1].RawHostPath == "/host/e2")
}

func TestDescription(t *testing.T) {
	config := ExperimentConfig{
		RawDescription: Description{
			RawString: ptrs.StringPtr("my_description"),
		},
	}

	// Test marshaling.
	bytes, err := json.Marshal(config)
	assert.NilError(t, err)

	rawObj := map[string]interface{}{}
	err = json.Unmarshal(bytes, &rawObj)
	assert.NilError(t, err)

	var expect interface{} = "my_description"
	assert.DeepEqual(t, rawObj["description"], expect)

	// Test unmarshaling.
	newConfig := ExperimentConfig{}
	err = json.Unmarshal(bytes, &newConfig)
	assert.NilError(t, err)

	assert.DeepEqual(t, newConfig.Description().String(), "my_description")
}

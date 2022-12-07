//nolint:exhaustivestruct
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
	out := schemas.Merge(e1, e2)
	assert.Assert(t, len(out.RawBindMounts) == 2)
	assert.Assert(t, out.RawBindMounts[0].RawHostPath == "/host/e1")
	assert.Assert(t, out.RawBindMounts[1].RawHostPath == "/host/e2")
}

func TestName(t *testing.T) {
	config := ExperimentConfig{
		RawName: Name{
			RawString: ptrs.Ptr("my_name"),
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

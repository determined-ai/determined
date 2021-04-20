package expconf

import (
	"testing"

	"gotest.tools/assert"

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

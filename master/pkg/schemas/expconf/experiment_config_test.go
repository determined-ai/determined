package expconf

import (
	"testing"

	"gotest.tools/assert"

	"github.com/determined-ai/determined/master/pkg/schemas"
)

func TestBindMountsMerge(t *testing.T) {
	e1 := ExperimentConfig{
		BindMounts: BindMountsConfig{
			BindMount{
				HostPath:      "/host/e1",
				ContainerPath: "/container/e1",
			},
		},
	}
	e2 := ExperimentConfig{
		BindMounts: BindMountsConfig{
			BindMount{
				HostPath:      "/host/e2",
				ContainerPath: "/container/e2",
			},
		},
	}
	out := schemas.Merge(e1, e2).(ExperimentConfig)
	assert.Assert(t, len(out.BindMounts) == 2)
	assert.Assert(t, out.BindMounts[0].HostPath == "/host/e1")
	assert.Assert(t, out.BindMounts[1].HostPath == "/host/e2")
}
